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
	Name         string `json:"name"`
	State        string `json:"state"`
	Reason       string `json:"reason"`
	Message      string `json:"message"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restartCount"`
}

// NotifyFunc represents callback function for Pod event
type NotifyFunc = func(event *PodEvent) error
