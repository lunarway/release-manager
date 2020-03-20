package apis

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

// Handle Flux events
func HandleV6(config APIConfig) (err error) {
	config.Server.HandleFunc("/v6/events", func(w http.ResponseWriter, r *http.Request) {
		config.Log.With("URL", r.URL).Info("Request for URL")

		eventStr, err := ioutil.ReadAll(r.Body)
		config.Log.With("EventStr", string(eventStr)).Info("Got flux event")

		event, err := ParseFluxEvent(bytes.NewBuffer(eventStr))
		if err != nil {
			config.Log.With("error", err.Error()).Error("got error parsing flux event")
			http.Error(w, err.Error(), 400)
			return
		}

		exporter := config.Exporter

		err = exporter.Send(r.Context(), config.Client, Message{
			Event: event,
		})
		if err != nil {
			config.Log.With("Error", err.Error()).Errorf("Exporter %T got an error", exporter)
			http.Error(w, err.Error(), 500)
			return
		}

		w.WriteHeader(200)
	})

	return nil
}
