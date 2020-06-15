package http

import (
	"context"
	"encoding/json"
	"io"

	"github.com/lunarway/release-manager/internal/tracing"
)

// payload is a struct tracing encoding and deconding operations of HTTP payloads.
type payload struct {
	tracer tracing.Tracer
}

// encodeResponse encodes resp as JSON into w. Tracing is reported from the
// context ctx and reported on tracer.
func (p *payload) encodeResponse(ctx context.Context, w io.Writer, resp interface{}) error {
	span, _ := p.tracer.FromCtx(ctx, "json encode response")
	defer span.Finish()
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		return err
	}
	return nil
}

// decodeResponse decodes req as JSON into r. Tracing is reported from the
// context ctx and reported on tracer.
func (p *payload) decodeResponse(ctx context.Context, r io.Reader, req interface{}) error {
	span, _ := p.tracer.FromCtx(ctx, "json decode request")
	defer span.Finish()
	decoder := json.NewDecoder(r)
	err := decoder.Decode(req)
	if err != nil {
		return err
	}
	return nil
}
