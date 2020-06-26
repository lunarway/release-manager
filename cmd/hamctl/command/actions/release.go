package actions

import (
	"net/http"

	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/intent"
)

func ReleaseArtifactID(client *httpinternal.Client, service, environment, artifactID string, intent intent.Intent) (httpinternal.ReleaseResponse, error) {
	var resp httpinternal.ReleaseResponse
	committerName, committerEmail, err := git.CommitterDetails()
	if err != nil {
		return resp, err
	}
	path, err := client.URL("release")
	if err != nil {
		return resp, err
	}
	err = client.Do(http.MethodPost, path, httpinternal.ReleaseRequest{
		Service:        service,
		Environment:    environment,
		ArtifactID:     artifactID,
		CommitterName:  committerName,
		CommitterEmail: committerEmail,
		Intent:         intent,
	}, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}
