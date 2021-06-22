package actions

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/lunarway/release-manager/generated/http/client"
	"github.com/lunarway/release-manager/generated/http/client/status"
)

func ArtifactIDFromEnvironment(client *client.ReleaseManagerServerAPI, clientAuth *runtime.ClientAuthInfoWriter, service, namespace, environment string) (string, error) {
	statusResp, err := client.Status.GetStatus(status.NewGetStatusParams().WithService(service).WithNamespace(&namespace), *clientAuth)
	if err != nil {
		return "", err
	}

	switch environment {
	case "dev":
		return statusResp.Payload.Dev.Tag, nil
	case "staging":
		return statusResp.Payload.Staging.Tag, nil
	case "prod":
		return statusResp.Payload.Prod.Tag, nil
	}

	return "", fmt.Errorf("unknown environment %s", environment)
}

func ArtifactIDFromBranch(client *client.ReleaseManagerServerAPI, clientAuth *runtime.ClientAuthInfoWriter, service string, branch string) (string, error) {
	describeResp, err := client.Status.GetDescribeArtifactService(
		status.NewGetDescribeArtifactServiceParams().
			WithBranch(&branch).
			WithService(service),
		*clientAuth,
	)
	if err != nil {
		return "", err
	}

	if len(describeResp.Payload.Artifacts) == 0 {
		return "", fmt.Errorf("no artifacts found on from branch '%s'", branch)
	}

	return describeResp.Payload.Artifacts[0].ID, nil
}
