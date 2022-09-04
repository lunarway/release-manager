package flux2notifications

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/lunarway/release-manager/internal/log"
)

func NewHttpServer(logger *log.Logger) *http.Server {
	router := mux.NewRouter()
	router.HandleFunc("/webhook/flux2-alerts", HandleEventFromFlux2(logger)).Methods(http.MethodPost)

	server := &http.Server{
		Addr:    ":3001",
		Handler: router,
	}

	return server
}
