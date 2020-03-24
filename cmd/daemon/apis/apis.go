package apis

import (
	"net/http"
	"os"
	"time"

	"github.com/lunarway/release-manager/internal/log"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
)

// All of the configuration necessary to run a fluxcloud API
type APIConfig struct {
	Server   *http.ServeMux
	Client   *http.Client
	Exporter Exporter
	// Config   Config
	Log *log.Logger
}

// Initialize API configuration
func NewAPIConfig(e Exporter, logger *log.Logger) APIConfig {
	return APIConfig{
		Server: http.NewServeMux(),
		Client: &http.Client{
			Timeout:   120 * time.Second,
			Transport: &ochttp.Transport{},
		},
		Exporter: e,
		Log:      logger,
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
