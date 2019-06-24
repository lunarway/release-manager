package command

import (
	"io"

	"github.com/lunarway/release-manager/internal/log"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-lib/metrics/prometheus"
)

func initTracing() (opentracing.Tracer, io.Closer, error) {
	cfg, err := config.FromEnv()
	if err != nil {
		return nil, nil, err
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = "release-manager"
	}
	cfg.Sampler = &config.SamplerConfig{
		Type:  jaeger.SamplerTypeConst,
		Param: 1,
	}
	cfg.Reporter.LogSpans = true
	log.WithFields("config", cfg).Infof("Tracing spans reported to '%s'", cfg.Reporter.LocalAgentHostPort)

	tracer, closer, err := cfg.NewTracer(
		config.Logger(&jaegerLogger{
			l: log.With("system", "jaeger"),
		}),
		config.Metrics(prometheus.New()),
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
