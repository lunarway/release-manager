package flux2notifications

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/lunarway/release-manager/internal/log"
)

type Event struct {
	InvolvedObject struct {
		Kind            string `json:"kind"`
		Namespace       string `json:"namespace"`
		Name            string `json:"name"`
		UID             string `json:"uid"`
		APIVersion      string `json:"apiVersion"`
		ResourceVersion string `json:"resourceVersion"`
	} `json:"involvedObject"`
	Severity            string    `json:"severity"`
	Timestamp           time.Time `json:"timestamp"`
	Message             string    `json:"message"`
	Reason              string    `json:"reason"`
	ReportingController string    `json:"reportingController"`
	ReportingInstance   string    `json:"reportingInstance"`
}

func HandleEventFromFlux2(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body) //TODO: log something
	if err != nil {
		log.Errorf("Failed to unmarshal alert from flux2-notification-controller: %v", err)
		http.Error(w, "unknown error", http.StatusInternalServerError)
		return
	}

	var event Event
	err = json.Unmarshal(body, &event)
	if err != nil {
		log.Errorf("Failed to unmarshal alert from flux2-notification-controller: %v", err)
		http.Error(w, "unknown error", http.StatusInternalServerError)
		return
	}
	log.Infof("Received alert from flux2-notification-controller: %s with msg: %s", event.InvolvedObject.Name, event.Message)
}
