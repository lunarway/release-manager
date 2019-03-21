package kubernetes

import (
	"time"
)

// PodEvent represents Pod termination event
type PodEvent struct {
	Namespace  string
	PodName    string
	StartedAt  time.Time
	FinishedAt time.Time
	ExitCode   int
	Reason     string
	Message    string
}

// NotifyFunc represents callback function for Pod event
type NotifyFunc = func(event *PodEvent) error
