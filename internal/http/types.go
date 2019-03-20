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
	Message string `json:"message,omitempty"`
}
