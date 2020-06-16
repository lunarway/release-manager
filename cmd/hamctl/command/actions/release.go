package actions

import (
	"fmt"
	"net/http"

	"github.com/lunarway/release-manager/internal/git"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/intent"
)

func ReleaseArtifactID(client *httpinternal.Client, service, environment, artifactID string, intent intent.Intent) error {
	committerName, committerEmail, err := git.CommitterDetails()
	if err != nil {
		return err
	}
	var resp httpinternal.ReleaseResponse
	path, err := client.URL("release")
	if err != nil {
		return err
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
		return err
	}
	fmt.Printf("Release of service: %s\n", service)
	if resp.Status != "" {
		fmt.Printf("%s\n", resp.Status)
	} else {
		fmt.Printf("[✓] Release of %s to %s initialized\n", resp.Tag, resp.ToEnvironment)
	}
	return nil
}
