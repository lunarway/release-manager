package actions

import (
	"net/http"

	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/intent"
)

type ReleaseResult struct {
	Response    httpinternal.ReleaseResponse
	Environment string
	Error       error
}

// ReleaseArtifactID issues a release request to a single environment.
func ReleaseArtifactID(client *httpinternal.Client, service, environment string, artifactID string, intent intent.Intent) (ReleaseResult, error) {
	resps, err := ReleaseArtifactIDMultipleEnvironments(client, service, []string{environment}, artifactID, intent)
	if err != nil {
		return ReleaseResult{}, err
	}
	return resps[0], nil
}

// ReleaseArtifactIDMultipleEnvironments issues a release request to multiple
// environments.
func ReleaseArtifactIDMultipleEnvironments(client *httpinternal.Client, service string, environments []string, artifactID string, intent intent.Intent) ([]ReleaseResult, error) {
	var results []ReleaseResult
	committerName, committerEmail, err := git.CommitterDetails()
	if err != nil {
		return nil, err
	}
	path, err := client.URL("release")
	if err != nil {
		return nil, err
	}
	for _, environment := range environments {
		var resp httpinternal.ReleaseResponse
		err = client.Do(http.MethodPost, path, httpinternal.ReleaseRequest{
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
