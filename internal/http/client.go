package http

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type Authenticator interface {
	Access(context context.Context) (*http.Client, error)
}

type Client struct {
	BaseURL  string
	Timeout  time.Duration
	Metadata Metadata
	Auth     Authenticator
}

type Metadata struct {
	CLIVersion string
}

// URL returns a URL with provided path added to the client's base URL.
func (c *Client) URL(path string) (string, error) {
	requestURL, err := url.Parse(fmt.Sprintf("%s/%s", c.BaseURL, path))
	if err != nil {
		return "", err
	}
	return requestURL.String(), nil
}

// URLWithQuery returns a URL with provided path and query params added to the
// client's base URL. All query param values are escaped.
func (c *Client) URLWithQuery(path string, queryParams url.Values) (string, error) {
	if queryParams != nil {
		path += fmt.Sprintf("?%s", queryParams.Encode())
	}
	return c.URL(path)
}

// Do sends an HTTP request defined by the provided method and path. The base
// URL is prefixed on the provided path.
//
// Request and response bodies are marshalled and unmarshalled as JSON and if
// the server returns a status code above 399 the response is parsed as an
// ErrorResponse object and returned as the error.
func (c *Client) Do(method string, path string, requestBody, responseBody interface{}) error {
	ctx := context.Background()
	client, err := c.Auth.Access(ctx)
	if err != nil {
		return err
	}
	client.Timeout = c.Timeout

	var b io.ReadWriter
	if requestBody != nil {
		b = &bytes.Buffer{}
		err := json.NewEncoder(b).Encode(requestBody)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequest(method, path, b)
	if err != nil {
		return err
	}
	id, err := uuid.NewRandom()
	if err == nil {
		req.Header.Set("x-request-id", id.String())
	}
	req.Header.Set("X-Cli-Version", c.Metadata.CLIVersion)
	resp, err := client.Do(req)
	if err != nil {
		var dnsError *net.DNSError
		var urlError *url.Error
		if stderrors.As(err, &dnsError) || (stderrors.As(err, &urlError) && stderrors.Is(err, io.EOF)) {
			return connectivityError
		}

		// Checking on the string as errors.Is fails to match on our ErrTokenExpired which is a copy of the internal oauth2 error from golang/x/oauth2 package.
		if strings.Contains(err.Error(), ErrTokenExpired.Error()) {
			return fmt.Errorf("%w: %w", ErrLoginRequired, err)
		}

		return &ErrorResponse{
			Message: err.Error(),
			ID:      id.String(),
		}
	}

	decoder := json.NewDecoder(resp.Body)
	if resp.StatusCode >= http.StatusBadRequest {
		var responseError ErrorResponse
		err = decoder.Decode(&responseError)
		if err != nil {
			var jsonError *json.SyntaxError
			if stderrors.As(err, &jsonError) {
				return connectivityError
			}
			return errors.WithMessagef(err, "response status %s: unmarshal error response", resp.Status)
		}
		responseError.ID = id.String()
		return &responseError
	}
	err = decoder.Decode(responseBody)
	if err != nil {
		return err
	}
	return nil
}

var connectivityError = &ErrorResponse{
	Message: "could not connect to the release-manager server. Are you connected to the internet and, if required, a VPN?",
}
