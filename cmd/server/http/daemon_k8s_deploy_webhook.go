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

func daemonk8sDeployWebhook(payload *payload, flowSvc *flow.Service, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// copy span from request context but ignore any deadlines on the request context
		ctx := opentracing.ContextWithSpan(context.Background(), opentracing.SpanFromContext(r.Context()))
		logger := logger.WithContext(ctx)
		var k8sReleaseEvent httpinternal.ReleaseEvent
		err := payload.decodeResponse(ctx, r.Body, &k8sReleaseEvent)
		if err != nil {
			logger.Errorf("http: daemon k8s deploy webhook: decode request body failed: %v", err)
			invalidBodyError(w, logger)
			return
		}
		logger = logger.WithFields("event", k8sReleaseEvent)
		err = flowSvc.NotifyK8SDeployEvent(ctx, &k8sReleaseEvent)
		if err != nil && errors.Cause(err) != slack.ErrUnknownEmail {
			logger.Errorf("http: daemon k8s deploy webhook failed: %+v", err)
		}
		w.WriteHeader(http.StatusOK)
		err = payload.encodeResponse(ctx, w, httpinternal.KubernetesNotifyResponse{})
		if err != nil {
			logger.Errorf("http: daemon k8s deploy webhook: environment: '%s' marshal response: %v", k8sReleaseEvent.Environment, err)
		}
		logger.Infof("http: daemon k8s deploy webhook: handled")
	}
}
