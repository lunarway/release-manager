package actions

import (
	"fmt"
	"net/http"
	"net/url"

	httpinternal "github.com/lunarway/release-manager/internal/http"
)

func ArtifactIDFromEnvironment(client *httpinternal.Client, service, namespace, environment string) (string, error) {
	var statusResp httpinternal.StatusResponse
	params := url.Values{}
	params.Add("service", service)
	if namespace != "" {
		params.Add("namespace", namespace)
	}
	path, err := client.URLWithQuery("status", params)
	if err != nil {
		return "", err
	}
	err = client.Do(http.MethodGet, path, nil, &statusResp)
	if err != nil {
		return "", err
	}

	for _, env := range statusResp.Environments {
		if environment == env.Name {
			return env.Tag, nil
		}
	}

	return "", fmt.Errorf("unknown environment %s", environment)
}

func ArtifactIDFromBranch(client *httpinternal.Client, service string, branch string) (string, error) {
	var describeResp httpinternal.DescribeArtifactResponse
	params := url.Values{}
	params.Add("branch", branch)
	path, err := client.URLWithQuery(fmt.Sprintf("describe/latest-artifact/%s/%s", service, branch), params)
	if err != nil {
		return "", err
	}
	err = client.Do(http.MethodGet, path, nil, &describeResp)
	if err != nil {
		return "", err
	}

	if len(describeResp.Artifacts) == 0 {
		return "", fmt.Errorf("no artifacts found on from branch '%s'", branch)
	}

	return describeResp.Artifacts[0].ID, nil
}
