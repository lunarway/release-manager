package tracing

import (
	"context"
	"fmt"
	"io"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-lib/metrics/prometheus"
)

// Tracer describes a tracing adapter interface.
type Tracer interface {
	io.Closer
	FromCtx(ctx context.Context, op string) (opentracing.Span, context.Context)
	FromCtxf(ctx context.Context, msg string, args ...interface{}) (opentracing.Span, context.Context)
}

type jaegerTracer struct {
	tracer  opentracing.Tracer
	flusher io.Closer
}

// NewJaeger allocates and returns a Jaeger implementation of the Tracer
// interface.
//
// It reads configuration from the environment and defaults to reporting spans
// to agents on localhost:6831. All spans are logged and Promethues metrics are
// registered on prometheus.DefaultRegisterer.
func NewJaeger() (Tracer, error) {
	cfg, err := config.FromEnv()
	if err != nil {
		return nil, err
	}
	cfg.ServiceName = "release-manager"
	cfg.Sampler = &config.SamplerConfig{
		Type:  jaeger.SamplerTypeConst,
		Param: 1,
	}
	cfg.Reporter.LogSpans = false
	log.WithFields("config", cfg).Infof("Tracing spans reported to '%s'", cfg.Reporter.LocalAgentHostPort)

	tracer, closer, err := cfg.NewTracer(
		config.Logger(&jaegerLogger{
			l: log.With("system", "jaeger"),
		}),
		config.Metrics(prometheus.New()),
	)
	if err != nil {
		return nil, err
	}

	return &jaegerTracer{
		tracer:  tracer,
		flusher: closer,
	}, nil
}

type jaegerLogger struct {
	l *log.Logger
}

func (j *jaegerLogger) Error(msg string) {
	j.l.Error(msg)
}

func (j *jaegerLogger) Infof(msg string, args ...interface{}) {
	j.l.Infof(msg, args...)
}

func (t *jaegerTracer) Close() error {
	return t.flusher.Close()
}

// FromCtx starts and returns a span with name `op` using a span found within
// the context `ctx` as a ChildOfRef. If that doesn't exist it creates a root
// span. It also returns a context.Context object built around the returned
// span.
func (t *jaegerTracer) FromCtx(ctx context.Context, op string) (opentracing.Span, context.Context) {
	return opentracing.StartSpanFromContextWithTracer(ctx, t.tracer, op)
}

// FromCtx starts and returns a span with a formatted name from `format` and
// `args` using a span found within the context `ctx` as a ChildOfRef. If that
// doesn't exist it creates a root span. It also returns a context.Context
// object built around the returned span.
func (t *jaegerTracer) FromCtxf(ctx context.Context, format string, args ...interface{}) (opentracing.Span, context.Context) {
	return t.FromCtx(ctx, fmt.Sprintf(format, args...))
}

// NewNoop allocates and returns a no-op implementation of the Tracer interface.
func NewNoop() Tracer {
	return &jaegerTracer{
		tracer:  &opentracing.NoopTracer{},
		flusher: &noopCloser{},
	}
}

type noopCloser struct {
	io.Reader
}

func (noopCloser) Close() error { return nil }
