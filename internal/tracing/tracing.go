package tracing

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/lunarway/release-manager/internal/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type requestIDKey struct{}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	v, ok := ctx.Value(requestIDKey{}).(string)
	if !ok {
		return ""
	}
	return v
}

// Tracer describes a tracing adapter interface.
type Tracer interface {
	io.Closer
	FromCtx(ctx context.Context, op string) (trace.Span, context.Context)
	FromCtxf(ctx context.Context, msg string, args ...interface{}) (trace.Span, context.Context)
}

type otelTracer struct {
	tracer   trace.Tracer
	provider *sdktrace.TracerProvider
}

// NewOTEL allocates and returns an OpenTelemetry implementation of the Tracer
// interface. It reads OTEL_EXPORTER_OTLP_ENDPOINT from the environment.
func NewOTEL() (Tracer, error) {
	ctx := context.Background()
	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create otlp exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceNameKey.String("release-manager")),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}
	log.Infof("Tracing spans reported to '%s'", endpoint)

	return &otelTracer{
		tracer:   provider.Tracer("release-manager"),
		provider: provider,
	}, nil
}

func (t *otelTracer) Close() error {
	if t.provider == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return t.provider.Shutdown(ctx)
}

// FromCtx starts and returns a span with name op using a span found within
// ctx as a parent. Also returns the context with the span embedded.
func (t *otelTracer) FromCtx(ctx context.Context, op string) (trace.Span, context.Context) {
	ctx, span := t.tracer.Start(ctx, op)
	return span, ctx
}

// FromCtxf starts and returns a span with a formatted name.
func (t *otelTracer) FromCtxf(ctx context.Context, format string, args ...interface{}) (trace.Span, context.Context) {
	return t.FromCtx(ctx, fmt.Sprintf(format, args...))
}

// NewNoop allocates and returns a no-op implementation of the Tracer interface.
func NewNoop() Tracer {
	return &otelTracer{
		tracer:   noop.NewTracerProvider().Tracer("noop"),
		provider: nil,
	}
}
