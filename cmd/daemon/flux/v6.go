package flux

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/lunarway/release-manager/internal/flux"
)

// HandleV6 Flux events
func HandleV6(api API) {
	api.Server.HandleFunc("/v6/events", func(w http.ResponseWriter, r *http.Request) {
		api.Log.With("url", r.URL).Info("Request for URL")

		defer r.Body.Close()
		event, err := ParseFluxEvent(r.Body)
		if err != nil {
			api.Log.With("error", err.Error()).Error("Error parsing flux event")
			http.Error(w, "Error parsing flux event", http.StatusInternalServerError)
			return
		}

		exporter := api.Exporter

		err = exporter.Send(r.Context(), event)
		if err != nil {
			api.Log.With("error", err.Error()).Errorf("Exporter %T got an error", exporter)
			http.Error(w, "Unknown error exporting the flux event", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

// ParseFluxEvent for doing flux event from Json into a flux Event struct.
func ParseFluxEvent(reader io.Reader) (flux.Event, error) {
	var evt flux.Event
	err := json.NewDecoder(reader).Decode(&evt)
	return evt, err
}
