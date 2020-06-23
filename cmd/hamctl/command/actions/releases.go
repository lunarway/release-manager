package actions

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	httpinternal "github.com/lunarway/release-manager/internal/http"
)

func ReleasesFromEnvironment(client *httpinternal.Client, service, environment string, count int) ([]httpinternal.DescribeReleaseResponseRelease, error) {
	var resp httpinternal.DescribeReleaseResponse
	params := url.Values{}
	params.Add("count", strconv.Itoa(count))
	path, err := client.URLWithQuery(fmt.Sprintf("describe/release/%s/%s", service, environment), params)
	if err != nil {
		return nil, err
	}
	err = client.Do(http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Releases, nil
}
