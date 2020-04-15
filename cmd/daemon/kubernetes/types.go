package kubernetes

// // PodEvent represents Pod termination event
// type PodEvent struct {
// 	Namespace      string      `json:"namespace"`
// 	Name           string      `json:"name"`
// 	State          string      `json:"state"`
// 	Reason         string      `json:"reason"`
// 	Message        string      `json:"message"`
// 	Containers     []Container `json:"containers"`
// 	ArtifactID     string      `json:"artifactId"`
// 	CommitterEmail string      `json:"committerEmail"`
// 	AuthorEmail    string      `json:"authorEmail"`
// }

// type Container struct {
// 	Name         string `json:"name"`
// 	State        string `json:"state"`
// 	Reason       string `json:"reason"`
// 	Message      string `json:"message"`
// 	Ready        bool   `json:"ready"`
// 	RestartCount int32  `json:"restartCount"`
// }

// type Log struct {
// 	Level   string
// 	Message string
// }

// // NotifyFunc represents callback function for Pod event
// type NotifyFunc = func(event *PodEvent) error

// //type ReleaseState string
// //
// //const (
// //	Released ReleaseState = "released"
// //	Deployed ReleaseState = "deployed"
// //	Failed   ReleaseState = "failed"
// //)

// type PodErrorEvent struct {
// 	Pod       string
// 	Container string
// }

type DeploymentEvent struct {
	Name        string
	Namespace   string
	Environment string
	Pods        []string
	ArtifactID  string
}

type CrashLoopBackOffEvent struct {
	PodName     string
	Namespace   string
	Environment string
	Logs        string
}

type CreateContainerConfigErrorEvent struct {
	PodName     string
	Namespace   string
	Environment string
	Error       string
}