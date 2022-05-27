package flux_notification_controller

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/lunarway/release-manager/internal/log"
)

func StartHttpServer() {
	router := mux.NewRouter()
	router.HandleFunc("/webhook/flux2-alerts", HandleEventFromFlux2).Methods(http.MethodPost)
	err := http.ListenAndServe(":3001", router)
	log.Errorf("Failed to start daemon's HTTP server: %v", err)
}
