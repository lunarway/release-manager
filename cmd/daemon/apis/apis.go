package apis

import (
	"net/http"
	"os"
	"time"

	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
)

// All of the configuration necessary to run a fluxcloud API
type APIConfig struct {
	Server    *http.ServeMux
	Client    *http.Client
	Exporter  []Exporter
	Formatter Formatter
	Config    Config
}

// Initialize API configuration
func NewAPIConfig(f Formatter, e []Exporter, c Config) APIConfig {
	return APIConfig{
		Server: http.NewServeMux(),
		Client: &http.Client{
			Timeout:   120 * time.Second,
			Transport: &ochttp.Transport{},
		},
		Formatter: f,
		Exporter:  e,
		Config:    c,
	}
}

// Listen on addr
func (a *APIConfig) Listen(addr string) error {
	if os.Getenv("JAEGER_ENDPOINT") != "" {
		exporter, err := jaeger.NewExporter(jaeger.Options{
			CollectorEndpoint: os.Getenv("JAEGER_ENDPOINT"),
			Process: jaeger.Process{
				ServiceName: "fluxcloud",
			},
		})
		if err != nil {
			return err
		}

		trace.RegisterExporter(exporter)
	}

	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	server := &http.Server{
		Addr:    addr,
		Handler: &ochttp.Handler{Handler: a.Server, IsPublicEndpoint: false},
	}

	return server.ListenAndServe()
}
