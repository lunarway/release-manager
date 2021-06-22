package actions

import (
	"github.com/go-openapi/runtime"
	"github.com/lunarway/release-manager/generated/http/client"
	"github.com/lunarway/release-manager/generated/http/client/status"
	"github.com/lunarway/release-manager/generated/http/models"
)

func ReleasesFromEnvironment(client *client.ReleaseManagerServerAPI, clientAuth *runtime.ClientAuthInfoWriter, service, environment string, count int64) (*models.DescribeReleaseResponse, error) {
	resp, err := client.Status.GetDescribeReleaseServiceEnvironment(
		status.NewGetDescribeReleaseServiceEnvironmentParams().
			WithCount(&count).
			WithEnvironment(environment).
			WithService(service),
		*clientAuth,
	)
	if err != nil {
		return nil, err
	}

	return resp.Payload, nil
}
