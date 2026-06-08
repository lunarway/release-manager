package http

import (
	"context"
	"net/http"

	"github.com/lunarway/release-manager/internal/flow"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/pkg/errors"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func daemonk8sPodErrorWebhook(payload *payload, flowSvc *flow.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// copy span from request context but ignore any deadlines on the request context
		ctx := oteltrace.ContextWithSpan(context.Background(), oteltrace.SpanFromContext(r.Context()))
		logger := log.WithContext(ctx)
		var event httpinternal.PodErrorEvent
		err := payload.decodeResponse(ctx, r.Body, &event)
		if err != nil {
			logger.Errorf("http: daemon k8s pod error webhook: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}
		logger = logger.WithFields("event", event)
		err = flowSvc.NotifyK8SPodErrorEvent(ctx, &event)
		if err != nil && errors.Cause(err) != slack.ErrUnknownEmail {
			logger.Errorf("http: daemon k8s pod error webhook failed: %+v", err)
		}
		w.WriteHeader(http.StatusOK)
		err = payload.encodeResponse(ctx, w, httpinternal.KubernetesNotifyResponse{})
		if err != nil {
			logger.Errorf("http: daemon k8s pod error webhook: environment: '%s' marshal response: %v", event.Environment, err)
		}
		logger.Infof("http: daemon k8s pod error webhook: handled")
	}
}
