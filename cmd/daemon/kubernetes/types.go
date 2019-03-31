package kubernetes

import (
	httpinternal "github.com/lunarway/release-manager/internal/http"
)

// PodEvent represents Pod termination event
type PodEvent struct {
	Namespace  string                   `json:"namespace"`
	Name       string                   `json:"name"`
	State      string                   `json:"state"`
	Reason     string                   `json:"reason"`
	Message    string                   `json:"message"`
	Containers []httpinternal.Container `json:"containers"`
	ArtifactID string                   `json:"artifactId"`
}

// NotifyFunc represents callback function for Pod event
type NotifyFunc = func(event *PodEvent) error
