package flux2notifications

import (
	"net/http"

	"github.com/gorilla/mux"
)

func NewHttpServer() *http.Server {
	router := mux.NewRouter()
	router.HandleFunc("/webhook/flux2-alerts", HandleEventFromFlux2).Methods(http.MethodPost)

	server := &http.Server{
		Addr:    ":3001",
		Handler: router,
	}

	return server
}
