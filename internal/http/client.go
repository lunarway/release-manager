package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

type Client struct {
	BaseURL   string
	Timeout   time.Duration
	AuthToken string
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

// Req executes an HTTP request defined by the provided method and path. The
// base URL is prefixed on the provided path.
//
// Request and response bodies are marshalled and unmarshalled as JSON and if
// the server returns a status code above 399 the response is parsed as an
// ErrorResponse object and the Message field is returned as an error.
func (c *Client) Req(method string, path string, requestBody, responseBody interface{}) error {
	client := &http.Client{
		Timeout: c.Timeout,
	}

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
	req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(resp.Body)
	if resp.StatusCode >= http.StatusBadRequest {
		var responseError ErrorResponse
		err = decoder.Decode(&responseError)
		if err != nil {
			return errors.WithMessagef(err, "response status %s: unmarshal error response", resp.Status)
		}
		return &responseError
	}
	err = decoder.Decode(responseBody)
	if err != nil {
		return err
	}
	return nil
}
