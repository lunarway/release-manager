package flux_notification_controller

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
	body, _ := ioutil.ReadAll(r.Body) //TODO: log something
	var event Event
	_ = json.Unmarshal(body, &event)
	log.Infof("Received alert from flux2-notification-controller: %s with msg: %s", event.InvolvedObject.Name, event.Message)
}
