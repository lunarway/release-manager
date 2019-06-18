package command

import (
	"io"

	"github.com/lunarway/release-manager/internal/log"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-lib/metrics"
)

func initTracing() (opentracing.Tracer, io.Closer, error) {
	cfg := config.Configuration{
		ServiceName: "release-manager",
		Sampler: &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans: true,
		},
	}

	// Example metrics factory. Use github.com/uber/jaeger-lib/metrics to bind to
	// real metrics frameworks.
	jMetricsFactory := metrics.NullFactory

	// Initialize tracer with a logger and a metrics factory
	tracer, closer, err := cfg.NewTracer(
		config.Logger(&jaegerLogger{
			l: log.With("system", "jaeger"),
		}),
		config.Metrics(jMetricsFactory),
	)
	if err != nil {
		return nil, nil, err
	}

	return tracer, closer, nil
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
