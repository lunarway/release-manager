package kubernetes

// PodEvent represents Pod termination event
type PodEvent struct {
	Namespace  string      `json:"namespace"`
	Name       string      `json:"name"`
	State      string      `json:"state"`
	Reason     string      `json:"reason"`
	Message    string      `json:"message"`
	Containers []Container `json:"containers"`
	ArtifactID string      `json:"artifactId"`
}

type Container struct {
	Name  string
	State string
}

// NotifyFunc represents callback function for Pod event
type NotifyFunc = func(event *PodEvent) error
