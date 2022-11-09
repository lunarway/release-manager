package events

import (
	"encoding/json"

	"github.com/lunarway/release-manager/internal/broker"
)

type ReleaseFailed struct {
	PodName     string   `json:"podName,omitempty"`
	Namespace   string   `json:"namespace,omitempty"`
	Errors      []string `json:"errors,omitempty"`
	AuthorEmail string   `json:"authorEmail,omitempty"`
	Environment string   `json:"environment,omitempty"`
	ArtifactID  string   `json:"artifactId,omitempty"`
	Squad       string   `json:"squad,omitempty"`
	AlertSquad  string   `json:"alertSquad,omitempty"`
}

// Marshal implements broker.Publishable
func (re ReleaseFailed) Marshal() ([]byte, error) {
	return json.Marshal(re)
}

// Unmarshal implements broker.Publishable
func (re ReleaseFailed) Unmarshal(data []byte) error {
	return json.Unmarshal(data, &re)
}

var _ broker.Publishable = &ReleaseFailed{}

func (re ReleaseFailed) Type() string {
	return "release_succeeded_event"
}
