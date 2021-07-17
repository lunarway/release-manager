package http

import (
	"net/http"
	"net/url"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	releasemanagerclient "github.com/lunarway/release-manager/generated/http/client"
)

type Config struct {
	BaseURL     string
	AuthToken   string
	CLIVersion  string
	CallerEmail string
}

func NewClient(config *Config) (*releasemanagerclient.ReleaseManagerServerAPI, runtime.ClientAuthInfoWriter, error) {
	url, err := url.ParseRequestURI(config.BaseURL)
	if err != nil {
		return nil, nil, err
	}
	transport := client.New(url.Host, "", []string{url.Scheme})

	transport.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		id, err := uuid.NewRandom()
		if err == nil {
			req.Header.Set("x-request-id", id.String())
		}
		if config.CLIVersion != "" {
			req.Header.Set("X-Cli-Version", config.CLIVersion)
		}
		if config.CallerEmail != "" {
			req.Header.Set("X-Caller-Email", config.CallerEmail)
		}
		return http.DefaultTransport.RoundTrip(req)
	})

	bearerTokenAuth := client.BearerToken(config.AuthToken)
	client := releasemanagerclient.New(transport, strfmt.Default)

	return client, bearerTokenAuth, err
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (r roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
}
