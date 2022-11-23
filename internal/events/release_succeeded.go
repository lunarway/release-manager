package events

import (
	"encoding/json"

	"github.com/lunarway/release-manager/internal/broker"
)

type ReleaseSucceeded struct {
	Name          string `json:"name,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	ResourceType  string `json:"resourceType,omitempty"`
	AvailablePods int32  `json:"availablePods,omitempty"`
	DesiredPods   int32  `json:"desiredPods,omitempty"`
	ArtifactID    string `json:"artifactId,omitempty"`
	AuthorEmail   string `json:"authorEmail,omitempty"`
	Environment   string `json:"environment,omitempty"`
}

// TODO: Make generic when go1.18 is adopted
// Marshal implements broker.Publishable
func (re ReleaseSucceeded) Marshal() ([]byte, error) {
	return json.Marshal(re)
}

// Unmarshal implements broker.Publishable
func (re ReleaseSucceeded) Unmarshal(data []byte) error {
	return json.Unmarshal(data, &re)
}

var _ broker.Publishable = &ReleaseSucceeded{}

func (re ReleaseSucceeded) Type() string {
	return "releaseSucceeded"
}
