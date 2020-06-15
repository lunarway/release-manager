package http

import (
	"context"
	"net/http"

	"github.com/lunarway/release-manager/internal/flow"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/slack"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
)

func daemonFluxWebhook(payload *payload, flowSvc *flow.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// copy span from request context but ignore any deadlines on the request context
		ctx := opentracing.ContextWithSpan(context.Background(), opentracing.SpanFromContext(r.Context()))
		logger := log.WithContext(ctx)
		var fluxNotifyEvent httpinternal.FluxNotifyRequest
		err := payload.decodeResponse(ctx, r.Body, &fluxNotifyEvent)
		if err != nil {
			logger.Errorf("http: daemon flux webhook: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}
		if !fluxNotifyEvent.Validate(w) {
			return
		}
		logger = logger.WithFields(
			"environment", fluxNotifyEvent.Environment,
			"event", fluxNotifyEvent.FluxEvent)

		err = flowSvc.NotifyFluxEvent(ctx, &fluxNotifyEvent)
		if err != nil && errors.Cause(err) != slack.ErrUnknownEmail {
			logger.Errorf("http: daemon flux webhook failed: %+v", err)
		}
		w.WriteHeader(http.StatusOK)
		err = payload.encodeResponse(ctx, w, httpinternal.FluxNotifyResponse{})
		if err != nil {
			logger.Errorf("http: daemon flux webhook: environment: '%s' marshal response: %v", fluxNotifyEvent.Environment, err)
		}
		logger.Infof("http: daemon flux webhook: handled")
	}
}
