package flux_notification_controller

import (
	"net/http"

	"github.com/gorilla/mux"
)

func StartHttpServer() {
	router := mux.NewRouter()
	router.HandleFunc("/webhook/flux2-alerts", HandleEventFromFlux2).Methods(http.MethodPost)
	_ = http.ListenAndServe(":3001", router) //TODO: log something
}
