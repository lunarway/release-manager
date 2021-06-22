package actions

import (
	"github.com/go-openapi/runtime"
	"github.com/lunarway/release-manager/generated/http/client"
	"github.com/lunarway/release-manager/generated/http/client/release"
	"github.com/lunarway/release-manager/generated/http/models"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/intent"
)

type ReleaseResult struct {
	Response    models.ReleaseResponse
	Environment string
	Error       error
}

// ReleaseArtifactID issues a release request to a single environment.
func ReleaseArtifactID(client *client.ReleaseManagerServerAPI, clientAuth *runtime.ClientAuthInfoWriter, service, environment string, artifactID string, intent intent.Intent) (ReleaseResult, error) {
	resps, err := ReleaseArtifactIDMultipleEnvironments(client, clientAuth, service, []string{environment}, artifactID, intent)
	if err != nil {
		return ReleaseResult{}, err
	}
	return resps[0], nil
}

// ReleaseArtifactIDMultipleEnvironments issues a release request to multiple
// environments.
func ReleaseArtifactIDMultipleEnvironments(client *client.ReleaseManagerServerAPI, clientAuth *runtime.ClientAuthInfoWriter, service string, environments []string, artifactID string, intent intent.Intent) ([]ReleaseResult, error) {
	var results []ReleaseResult
	committerName, committerEmail, err := git.CommitterDetails()
	if err != nil {
		return nil, err
	}
	for _, environment := range environments {
		resp, err := client.Release.PostRelease(release.NewPostReleaseParams().WithBody(&models.ReleaseRequest{
			Service:        &service,
			Environment:    &environment,
			ArtifactID:     &artifactID,
			CommitterName:  &committerName,
			CommitterEmail: &committerEmail,
			Intent: &models.Intent{
				Type: &intent.Type,
				Promote: &models.IntentPromote{
					FromEnvironment: intent.Promote.FromEnvironment,
				},
				ReleaseBranch: &models.IntentReleaseBranch{
					Branch: intent.ReleaseBranch.Branch,
				},
				Rollback: &models.IntentRollback{
					PreviousArtifactID: intent.Rollback.PreviousArtifactID,
				},
			},
		}), *clientAuth)
		if err != nil {
			return nil, err
		}

		results = append(results, ReleaseResult{
			Response:    *resp.Payload,
			Environment: environment,
			Error:       err,
		})
	}
	return results, nil
}
