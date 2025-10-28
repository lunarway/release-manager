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

func daemonk8sJobErrorWebhook(payload *payload, flowSvc *flow.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// copy span from request context but ignore any deadlines on the request context
		ctx := opentracing.ContextWithSpan(context.Background(), opentracing.SpanFromContext(r.Context()))
		logger := log.WithContext(ctx)
		var event httpinternal.JobErrorEvent
		err := payload.decodeResponse(ctx, r.Body, &event)
		if err != nil {
			logger.Errorf("http: daemon k8s job error webhook: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}
		logger = logger.WithFields("event", event)
		err = flowSvc.NotifyK8SJobErrorEvent(ctx, &event)
		if err != nil && errors.Cause(err) != slack.ErrUnknownEmail {
			logger.Errorf("http: daemon k8s job error webhook failed: %+v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = payload.encodeResponse(ctx, w, httpinternal.KubernetesNotifyResponse{})
		if err != nil {
			logger.Errorf("http: daemon k8s job error webhook: environment: '%s' marshal response: %v", event.Environment, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		logger.Infof("http: daemon k8s job error webhook: handled")
	}
}
