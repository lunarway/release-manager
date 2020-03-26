package flux

import (
	"net/http"

	"github.com/lunarway/release-manager/internal/log"
	"go.opencensus.io/plugin/ochttp"
)

// API has the configuration necessary to run a flux API
type API struct {
	Server   *http.ServeMux
	Exporter Exporter
	Log      *log.Logger
}

// NewAPI initialize API configuration
func NewAPI(e Exporter, logger *log.Logger) API {
	return API{
		Server:   http.NewServeMux(),
		Exporter: e,
		Log:      logger,
	}
}

// Listen on addr
func (a *API) Listen(addr string) error {
	server := &http.Server{
		Addr:    addr,
		Handler: &ochttp.Handler{Handler: a.Server, IsPublicEndpoint: false},
	}

	return server.ListenAndServe()
}
