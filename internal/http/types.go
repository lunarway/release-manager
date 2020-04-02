package http

import (
	"fmt"
	"time"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/weaveworks/flux"
	"github.com/weaveworks/flux/event"
	"github.com/weaveworks/flux/update"
)

type StatusRequest struct {
	Service string `json:"service,omitempty"`
}

type StatusResponse struct {
	DefaultNamespaces bool         `json:"defaultNamespaces,omitempty"`
	Dev               *Environment `json:"dev,omitempty"`
	Staging           *Environment `json:"staging,omitempty"`
	Prod              *Environment `json:"prod,omitempty"`
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
	Namespace      string `json:"namespace,omitempty"`
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
	ID      string `json:"-"`
}

var _ error = &ErrorResponse{}

func (e *ErrorResponse) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s\nReference: %s", e.Message, e.ID)
	}
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

type FluxNotifyResponse struct {
	Status string `json:"status,omitempty"`
}

type FluxNotifyRequest struct {
	Environment        string `json:"environment,omitempty"`
	EventID            event.EventID
	EventServiceIDs    []flux.ResourceID
	EventChangedImages []string
	EventResult        update.Result
	EventType          string
	EventStartedAt     time.Time
	EventEndedAt       time.Time
	EventLogLevel      string
	EventMessage       string
	EventString        string
	Commits            []event.Commit
	Errors             []event.ResourceError
}

type PodNotifyResponse struct {
	Status string `json:"status,omitempty"`
}

type PodNotifyRequest struct {
	Namespace      string      `json:"namespace"`
	Name           string      `json:"name"`
	State          string      `json:"state"`
	Reason         string      `json:"reason"`
	Message        string      `json:"message"`
	Containers     []Container `json:"containers"`
	ArtifactID     string      `json:"artifactId"`
	Logs           string      `json:"logs"`
	Environment    string      `json:"environment"`
	CommitterEmail string      `json:"committerEmail"`
	AuthorEmail    string      `json:"authorEmail"`
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

type RollbackRequest struct {
	Service        string `json:"service,omitempty"`
	Namespace      string `json:"namespace,omitempty"`
	Environment    string `json:"environment,omitempty"`
	CommitterName  string `json:"committerName,omitempty"`
	CommitterEmail string `json:"committerEmail,omitempty"`
}

type RollbackResponse struct {
	Service            string `json:"service,omitempty"`
	Status             string `json:"status,omitempty"`
	Environment        string `json:"environment,omitempty"`
	PreviousArtifactID string `json:"previousArtifactId,omitempty"`
	NewArtifactID      string `json:"newArtifactId,omitempty"`
}

type DescribeReleaseResponse struct {
	Service         string        `json:"service,omitempty"`
	Environment     string        `json:"environment,omitempty"`
	Artifact        artifact.Spec `json:"artifact,omitempty"`
	ReleasedAt      time.Time     `json:"releasedAt,omitempty"`
	ReleasedByEmail string        `json:"releasedByEmail,omitempty"`
	ReleasedByName  string        `json:"releasedByName,omitempty"`
}

type DescribeArtifactResponse struct {
	Service   string          `json:"service,omitempty"`
	Artifacts []artifact.Spec `json:"artifacts,omitempty"`
}
