package actions

import (
	"net/http"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/intent"
)

//go:generate moq -out config_mock.go . GitConfigAPI

// GitConfigAPI is an interface to interact with a git config system
// this makes it possible to extract information from the repository
// or the local user
type GitConfigAPI interface {
	CommitterDetails() (name string, email string, err error)
}

type ReleaseResult struct {
	Response    httpinternal.ReleaseResponse
	Environment string
	Error       error
}

type ReleaseHttpClient struct {
	gitConfigAPI GitConfigAPI
	client       *httpinternal.Client
}

func NewReleaseHttpClient(gitConfigAPI GitConfigAPI, client *httpinternal.Client) *ReleaseHttpClient {
	return &ReleaseHttpClient{
		gitConfigAPI: gitConfigAPI,
		client:       client,
	}
}

// ReleaseArtifactID issues a release request to a single environment.
func (hc *ReleaseHttpClient) ReleaseArtifactID(service, environment string, artifactID string, intent intent.Intent) (ReleaseResult, error) {
	resps, err := hc.ReleaseArtifactIDMultipleEnvironments(service, []string{environment}, artifactID, intent)
	if err != nil {
		return ReleaseResult{}, err
	}
	return resps[0], nil
}

// ReleaseArtifactIDMultipleEnvironments issues a release request to multiple
// environments.
func (hc *ReleaseHttpClient) ReleaseArtifactIDMultipleEnvironments(service string, environments []string, artifactID string, intent intent.Intent) ([]ReleaseResult, error) {
	var results []ReleaseResult
	committerName, committerEmail, err := hc.gitConfigAPI.CommitterDetails()
	if err != nil {
		return nil, err
	}
	path, err := hc.client.URL("release")
	if err != nil {
		return nil, err
	}
	for _, environment := range environments {
		var resp httpinternal.ReleaseResponse
		err = hc.client.Do(http.MethodPost, path, httpinternal.ReleaseRequest{
			Service:        service,
			Environment:    environment,
			ArtifactID:     artifactID,
			CommitterName:  committerName,
			CommitterEmail: committerEmail,
			Intent:         intent,
		}, &resp)

		results = append(results, ReleaseResult{
			Response:    resp,
			Environment: environment,
			Error:       err,
		})
	}
	return results, nil
}
