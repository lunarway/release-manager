package actions

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/lunarway/release-manager/internal/artifact"
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

// LatestArtifactFromBranch returns the latest artifact of service built from
// branch.
func LatestArtifactFromBranch(client *httpinternal.Client, service string, branch string) (artifact.Spec, error) {
	var describeResp httpinternal.DescribeArtifactResponse
	params := url.Values{}
	params.Add("branch", branch)
	path, err := client.URLWithQuery("describe/latest-artifact/"+service, params)
	if err != nil {
		return artifact.Spec{}, err
	}
	err = client.Do(http.MethodGet, path, nil, &describeResp)
	if err != nil {
		return artifact.Spec{}, err
	}

	if len(describeResp.Artifacts) == 0 {
		return artifact.Spec{}, fmt.Errorf("no artifacts found on from branch '%s'", branch)
	}

	return describeResp.Artifacts[0], nil
}

// ArtifactsFromBranch returns up to count of the newest artifacts of service
// built from branch, newest first. The list is empty if there are none.
func ArtifactsFromBranch(client *httpinternal.Client, service, branch string, count int) ([]artifact.Spec, error) {
	var describeResp httpinternal.DescribeArtifactResponse
	params := url.Values{}
	// Artifacts record their branch with slashes replaced by underscores and
	// the server compares it exactly.
	params.Add("branch", strings.ReplaceAll(branch, "/", "_"))
	params.Add("count", strconv.Itoa(count))
	path, err := client.URLWithQuery("describe/artifact/"+service, params)
	if err != nil {
		return nil, err
	}
	err = client.Do(http.MethodGet, path, nil, &describeResp)
	if err != nil {
		return nil, err
	}

	return describeResp.Artifacts, nil
}
