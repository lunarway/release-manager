package actions

import (
	"net/http"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/intent"
)

type ReleaseResult struct {
	Response    httpinternal.ReleaseResponse
	Environment string
	Error       error
}

type ReleaseHttpClient struct {
	client *httpinternal.Client
}

func NewReleaseHttpClient(client *httpinternal.Client) *ReleaseHttpClient {
	return &ReleaseHttpClient{
		client: client,
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
	path, err := hc.client.URL("release")
	if err != nil {
		return nil, err
	}
	for _, environment := range environments {
		var resp httpinternal.ReleaseResponse
		err = hc.client.Do(http.MethodPost, path, httpinternal.ReleaseRequest{
			Service:     service,
			Environment: environment,
			ArtifactID:  artifactID,
			Intent:      intent,
		}, &resp)

		results = append(results, ReleaseResult{
			Response:    resp,
			Environment: environment,
			Error:       err,
		})
	}
	return results, nil
}
