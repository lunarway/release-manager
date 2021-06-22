package http

import (
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	releasemanagerclient "github.com/lunarway/release-manager/generated/http/client"
)

func NewClient(baseURL, token string, timeout time.Duration) (*releasemanagerclient.ReleaseManagerServerAPI, runtime.ClientAuthInfoWriter) {
	transport := client.New(baseURL, "", nil)
	transport.Transport = NewRoundTripper(timeout, "", "")

	bearerTokenAuth := client.BearerToken(token)
	client := releasemanagerclient.New(transport, strfmt.Default)

	return client, bearerTokenAuth
}
