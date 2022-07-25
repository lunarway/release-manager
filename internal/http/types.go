package http

import (
	"net/http"
	"time"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/intent"
)

type StatusRequest struct {
	Service string `json:"service,omitempty"`
}

type StatusResponse struct {
	DefaultNamespaces bool          `json:"defaultNamespaces,omitempty"`
	Environments      []Environment `json:"environments,omitempty"`
}

type Environment struct {
	Name                  string `json:"name,omitempty"`
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

type ReleaseRequest struct {
	Service        string        `json:"service,omitempty"`
	Environment    string        `json:"environment,omitempty"`
	ArtifactID     string        `json:"artifactId,omitempty"`
	CommitterName  string        `json:"committerName,omitempty"`
	CommitterEmail string        `json:"committerEmail,omitempty"`
	Intent         intent.Intent `json:"intent,omitempty"`
}

func (r ReleaseRequest) Validate(w http.ResponseWriter) bool {
	var errs validationErrors
	if emptyString(r.Service) {
		errs.Append(requiredField("service"))
	}
	if emptyString(r.Environment) {
		errs.Append(requiredField("environment"))
	}
	if emptyString(r.CommitterName) {
		errs.Append(requiredField("committerName"))
	}
	if emptyString(r.CommitterEmail) {
		errs.Append(requiredField("committerEmail"))
	}
	if emptyString(r.ArtifactID) {
		errs.Append("required field artifact id is not specified")
	}
	if r.Intent.Empty() {
		errs.Append("required intent is not specified")
	}
	if !r.Intent.Valid() {
		errs.Append("required intent is not valid")
	}
	return errs.Evaluate(w)
}

type ReleaseResponse struct {
	Service       string `json:"service,omitempty"`
	ReleaseID     string `json:"releaseId,omitempty"`
	Status        string `json:"status,omitempty"`
	ToEnvironment string `json:"toEnvironment,omitempty"`
	Tag           string `json:"tag,omitempty"`
}

type ReleaseEvent struct {
	Name          string `json:"name,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	ResourceType  string `json:"resourceType,omitempty"`
	AvailablePods int32  `json:"availablePods"`
	DesiredPods   int32  `json:"replicas,omitempty"`
	ArtifactID    string `json:"artifactId,omitempty"`
	AuthorEmail   string `json:"authorEmail,omitempty"`
	Environment   string `json:"environment,omitempty"`
}
type ContainerError struct {
	Name         string `json:"name,omitempty"`
	ErrorMessage string `json:"message,omitempty"`
	Type         string `json:"type,omitempty"`
}

type PodErrorEvent struct {
	PodName     string           `json:"podName,omitempty"`
	Namespace   string           `json:"namespace,omitempty"`
	Errors      []ContainerError `json:"errors,omitempty"`
	AuthorEmail string           `json:"authorEmail,omitempty"`
	Environment string           `json:"environment,omitempty"`
	ArtifactID  string           `json:"artifactId,omitempty"`
	Squad       string           `json:"squad,omitempty"`
	AlertSquad  string           `json:"alertSquad,omitempty"`
}

type JobConditionError struct {
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type JobErrorEvent struct {
	JobName     string              `json:"jobName,omitempty"`
	Namespace   string              `json:"namespace,omitempty"`
	Errors      []JobConditionError `json:"errors,omitempty"`
	AuthorEmail string              `json:"authorEmail,omitempty"`
	Environment string              `json:"environment,omitempty"`
	ArtifactID  string              `json:"artifactId,omitempty"`
}

type KubernetesNotifyResponse struct {
}

type ListPoliciesResponse struct {
	Service            string                    `json:"service,omitempty"`
	AutoReleases       []AutoReleasePolicy       `json:"autoReleases,omitempty"`
	BranchRestrictions []BranchRestrictionPolicy `json:"branchRestrictions,omitempty"`
}

type AutoReleasePolicy struct {
	ID          string `json:"id,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Environment string `json:"environment,omitempty"`
}

type BranchRestrictionPolicy struct {
	ID          string `json:"id,omitempty"`
	Environment string `json:"environment,omitempty"`
	BranchRegex string `json:"branchRegex,omitempty"`
}

type ApplyBranchRestrictionPolicyRequest struct {
	Service        string `json:"service,omitempty"`
	Environment    string `json:"environment,omitempty"`
	BranchRegex    string `json:"branchRegex,omitempty"`
	CommitterName  string `json:"committerName,omitempty"`
	CommitterEmail string `json:"committerEmail,omitempty"`
}

func (r ApplyBranchRestrictionPolicyRequest) Validate(w http.ResponseWriter) bool {
	var errs validationErrors
	if emptyString(r.Service) {
		errs.Append(requiredField("service"))
	}
	if emptyString(r.Environment) {
		errs.Append(requiredField("environment"))
	}
	if emptyString(r.BranchRegex) {
		errs.Append(requiredField("branch regex"))
	}
	if emptyString(r.CommitterName) {
		errs.Append(requiredField("committerName"))
	}
	if emptyString(r.CommitterEmail) {
		errs.Append(requiredField("committerEmail"))
	}
	return errs.Evaluate(w)
}

type ApplyBranchRestrictionPolicyResponse struct {
	ID          string `json:"id,omitempty"`
	Service     string `json:"service,omitempty"`
	Environment string `json:"environment,omitempty"`
	BranchRegex string `json:"branchRegex,omitempty"`
}

type ApplyAutoReleasePolicyRequest struct {
	Service        string `json:"service,omitempty"`
	Branch         string `json:"branch,omitempty"`
	Environment    string `json:"environment,omitempty"`
	CommitterName  string `json:"committerName,omitempty"`
	CommitterEmail string `json:"committerEmail,omitempty"`
}

func (r ApplyAutoReleasePolicyRequest) Validate(w http.ResponseWriter) bool {
	var errs validationErrors
	if emptyString(r.Service) {
		errs.Append(requiredField("service"))
	}
	if emptyString(r.Branch) {
		errs.Append(requiredField("branch"))
	}
	if emptyString(r.Environment) {
		errs.Append(requiredField("environment"))
	}
	if emptyString(r.CommitterName) {
		errs.Append(requiredField("committerName"))
	}
	if emptyString(r.CommitterEmail) {
		errs.Append(requiredField("committerEmail"))
	}
	return errs.Evaluate(w)
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

func (r DeletePolicyRequest) Validate(w http.ResponseWriter) bool {
	var errs validationErrors
	if emptyString(r.Service) {
		errs.Append(requiredField("service"))
	}
	if emptyString(r.CommitterName) {
		errs.Append(requiredField("committerName"))
	}
	if emptyString(r.CommitterEmail) {
		errs.Append(requiredField("committerEmail"))
	}
	ids := filterEmptyStrings(r.PolicyIDs)
	if len(ids) == 0 {
		errs.Append("no policy ids suplied")
	}
	return errs.Evaluate(w)
}

type DeletePolicyResponse struct {
	Service string `json:"service,omitempty"`
	Count   int    `json:"count,omitempty"`
}

// DescribeReleaseResponse returns releases for a service. The Releases returned are in
// chronically order with the latest and current release first
type DescribeReleaseResponse struct {
	Service     string                           `json:"service,omitempty"`
	Environment string                           `json:"environment,omitempty"`
	Releases    []DescribeReleaseResponseRelease `json:"releases,omitempty"`
}

type DescribeReleaseResponseRelease struct {
	ReleaseIndex    int           `json:"releaseIndex,omitempty"`
	Artifact        artifact.Spec `json:"artifact,omitempty"`
	ReleasedAt      time.Time     `json:"releasedAt,omitempty"`
	ReleasedByEmail string        `json:"releasedByEmail,omitempty"`
	ReleasedByName  string        `json:"releasedByName,omitempty"`
	Intent          intent.Intent `json:"intent,omitempty"`
}

type DescribeArtifactResponse struct {
	Service   string          `json:"service,omitempty"`
	Artifacts []artifact.Spec `json:"artifacts,omitempty"`
}

type ArtifactUploadRequest struct {
	Artifact artifact.Spec `json:"artifact,omitempty"`
	MD5      string        `json:"md5,omitempty"`
}

func (r ArtifactUploadRequest) Validate(w http.ResponseWriter) bool {
	var errs validationErrors
	if emptyString(r.MD5) {
		errs.Append(requiredField("md5"))
	}
	if emptyString(r.Artifact.ID) {
		errs.Append(requiredField("artifact.id"))
	}
	if emptyString(r.Artifact.Service) {
		errs.Append(requiredField("artifact.service"))
	}
	return errs.Evaluate(w)
}

type ArtifactUploadResponse struct {
	ArtifactUploadURL string `json:"artifactUploadUrl,omitempty"`
}
