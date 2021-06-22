package http

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

var _ http.RoundTripper = &RoundTripper{}

type RoundTripper struct {
	underlyingTransport http.RoundTripper
	timeout             time.Duration
	cliVersion          string
	callerEmail         string
}

func NewRoundTripper(timeout time.Duration, cliVersion, callerEmail string) *RoundTripper {
	return &RoundTripper{
		underlyingTransport: http.DefaultTransport,
		timeout:             timeout,
		cliVersion:          cliVersion,
		callerEmail:         callerEmail,
	}
}

func (r *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), r.timeout)
	defer cancel()
	*req = *req.WithContext(ctx)

	id, err := uuid.NewRandom()
	if err == nil {
		req.Header.Set("x-request-id", id.String())
	}
	if r.cliVersion != "" {
		req.Header.Set("X-Cli-Version", r.cliVersion)
	}
	if r.callerEmail != "" {
		req.Header.Set("X-Caller-Email", r.callerEmail)
	}
	return r.underlyingTransport.RoundTrip(req)
}
