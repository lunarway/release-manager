package kubernetes

// PodEvent represents Pod termination event
type PodEvent struct {
	Namespace  string `json:"namespace"`
	PodName    string `json:"podName"`
	Status     string `json:"status"`
	Reason     string `json:"reason"`
	Message    string `json:"message"`
	ArtifactID string `json:"artifactId"`
}

// NotifyFunc represents callback function for Pod event
type NotifyFunc = func(event *PodEvent) error
