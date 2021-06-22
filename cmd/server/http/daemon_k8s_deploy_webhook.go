package http

import (
	"context"

	"github.com/go-openapi/runtime/middleware"
	"github.com/lunarway/release-manager/generated/http/models"
	"github.com/lunarway/release-manager/generated/http/restapi/operations"
	"github.com/lunarway/release-manager/generated/http/restapi/operations/webhook"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/slack"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
)

func Daemonk8sDeployWebhookHandler(flowSvc *flow.Service) HandlerFactory {
	return func(api *operations.ReleaseManagerServerAPIAPI) {
		// copy span from request context but ignore any deadlines on the request context
		api.WebhookPostWebhookDaemonK8sDeployHandler = webhook.PostWebhookDaemonK8sDeployHandlerFunc(func(params webhook.PostWebhookDaemonK8sDeployParams, principal interface{}) middleware.Responder {
			ctx := opentracing.ContextWithSpan(context.Background(), opentracing.SpanFromContext(params.HTTPRequest.Context()))
			logger := log.WithContext(ctx)

			event := params.Body

			logger = logger.WithFields("event", event)
			err := flowSvc.NotifyK8SDeployEvent(ctx, event)
			if err != nil && errors.Cause(err) != slack.ErrUnknownEmail {
				logger.Errorf("http: daemon k8s deploy webhook failed: %+v", err)
			}

			logger.Infof("http: daemon k8s deploy webhook: handled")
			return webhook.NewPostWebhookDaemonK8sDeployOK().WithPayload(models.EmptyWebhookResponse(struct{}{}))
		})
	}
}
