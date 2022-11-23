package events

import (
	"encoding/json"

	"github.com/lunarway/release-manager/internal/broker"
)

type ReleasedEvent struct {
	Service     string `json:"name,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	ArtifactID  string `json:"artifactId,omitempty"`
	AuthorEmail string `json:"authorEmail,omitempty"`
	AuthorName  string `json:"authorName,omitempty"`
	Squad       string `json:"squad,omitempty"`
	Environment string `json:"environment,omitempty"`
	IntentType  string `json:"intentType,omitempty"`
}

// TODO: Make generic when go1.18 is adopted
// Marshal implements broker.Publishable
func (re ReleasedEvent) Marshal() ([]byte, error) {
	return json.Marshal(re)
}

// Unmarshal implements broker.Publishable
func (re ReleasedEvent) Unmarshal(data []byte) error {
	return json.Unmarshal(data, &re)
}

var _ broker.Publishable = &ReleasedEvent{}

func (re ReleasedEvent) Type() string {
	return "released"
}
