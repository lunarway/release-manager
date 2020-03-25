package flux

import (
	"net/http"
)

// Handle Flux events
func HandleV6(api API) (err error) {
	api.Server.HandleFunc("/v6/events", func(w http.ResponseWriter, r *http.Request) {
		api.Log.With("URL", r.URL).Info("Request for URL")

		defer r.Body.Close()
		event, err := ParseFluxEvent(r.Body)
		if err != nil {
			api.Log.With("error", err.Error()).Error("Error parsing flux event")
			http.Error(w, "Error parsing flux event", http.StatusInternalServerError)
			return
		}

		exporter := api.Exporter

		err = exporter.Send(r.Context(), Message{
			Event: event,
		})
		if err != nil {
			api.Log.With("Error", err.Error()).Errorf("Exporter %T got an error", exporter)
			http.Error(w, "Unknown error exporting the flux event", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	return nil
}
