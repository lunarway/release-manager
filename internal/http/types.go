package http

type StatusRequest struct {
	Service string `json:"service,omitempty"`
}

type StatusResponse struct {
	Dev     *Environment `json:"dev,omitempty"`
	Staging *Environment `json:"staging,omitempty"`
	Prod    *Environment `json:"prod,omitempty"`
}

type Environment struct {
	Tag                   string `json:"tag,omitempty"`
	Committer             string `json:"committer,omitempty"`
	Author                string `json:"author,omitempty"`
	Message               string `json:"message,omitempty"`
	Date                  int64  `json:"date,omitempty"`
	BuildUrl              string `json:"buildUrl,omitempty"`
	HighVulnerabilities   int64  `json:"highVulnerabilities,omitempty"`
	MediumVulnerabilities int64  `json:"mediumVulnerabilities,omitempty"`
	LowVulnerabilities    int64  `json:"lowVulnerabilities,omitempty"`
}

type PromoteRequest struct {
	Service        string `json:"service,omitempty"`
	Environment    string `json:"environment,omitempty"`
	CommitterName  string `json:"committerName,omitempty"`
	CommitterEmail string `json:"committerEmail,omitempty"`
}

type PromoteResponse struct {
	Service         string `json:"service,omitempty"`
	FromEnvironment string `json:"fromEnvironment,omitempty"`
	Status          string `json:"status,omitempty"`
	ToEnvironment   string `json:"toEnvironment,omitempty"`
	Tag             string `json:"tag,omitempty"`
}

type ErrorResponse struct {
	Status  int    `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

var _ error = &ErrorResponse{}

func (e *ErrorResponse) Error() string {
	return e.Message
}

type ReleaseRequest struct {
	Service        string `json:"service,omitempty"`
	Environment    string `json:"environment,omitempty"`
	Branch         string `json:"branch,omitempty"`
	ArtifactID     string `json:"artifactId,omitempty"`
	CommitterName  string `json:"committerName,omitempty"`
	CommitterEmail string `json:"committerEmail,omitempty"`
}

type ReleaseResponse struct {
	Service       string `json:"service,omitempty"`
	ReleaseID     string `json:"releaseId,omitempty"`
	Status        string `json:"status,omitempty"`
	ToEnvironment string `json:"toEnvironment,omitempty"`
	Tag           string `json:"tag,omitempty"`
}

type PodNotifyRequest struct {
	Namespace  string      `json:"namespace"`
	Name       string      `json:"name"`
	State      string      `json:"state"`
	Reason     string      `json:"reason"`
	Message    string      `json:"message"`
	Containers []Container `json:"containers"`
	ArtifactID string      `json:"artifactId"`
	Logs       string      `json:"logs"`
}
type Container struct {
	Name         string `json:"name"`
	State        string `json:"state"`
	Reason       string `json:"reason"`
	Message      string `json:"message"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restartCount"`
}

type ListPoliciesResponse struct {
	Service      string              `json:"service,omitempty"`
	AutoReleases []AutoReleasePolicy `json:"autoReleases,omitempty"`
}

type AutoReleasePolicy struct {
	ID          string `json:"id,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Environment string `json:"environment,omitempty"`
}

type ApplyAutoReleasePolicyRequest struct {
	Service        string `json:"service,omitempty"`
	Branch         string `json:"branch,omitempty"`
	Environment    string `json:"environment,omitempty"`
	CommitterName  string `json:"committerName,omitempty"`
	CommitterEmail string `json:"committerEmail,omitempty"`
}

type ApplyPolicyResponse struct {
	ID          string `json:"id,omitempty"`
	Service     string `json:"service,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Environment string `json:"environment,omitempty"`
}

type DeletePolicyRequest struct {
	Service        string   `json:"service,omitempty"`
	PolicyIDs      []string `json:"policyIds,omitempty"`
	CommitterName  string   `json:"committerName,omitempty"`
	CommitterEmail string   `json:"committerEmail,omitempty"`
}

type DeletePolicyResponse struct {
	Service string `json:"service,omitempty"`
	Count   int    `json:"count,omitempty"`
}
