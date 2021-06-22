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

func Daemonk8sJobErrorWebhookHandler(flowSvc *flow.Service) HandlerFactory {
	return func(api *operations.ReleaseManagerServerAPIAPI) {
		api.WebhookPostWebhookDaemonK8sJoberrorHandler = webhook.PostWebhookDaemonK8sJoberrorHandlerFunc(func(params webhook.PostWebhookDaemonK8sJoberrorParams, principal interface{}) middleware.Responder {
			// copy span from request context but ignore any deadlines on the request context
			ctx := opentracing.ContextWithSpan(context.Background(), opentracing.SpanFromContext(params.HTTPRequest.Context()))
			logger := log.WithContext(ctx)

			event := params.Body

			logger = logger.WithFields("event", event)

			err := flowSvc.NotifyK8SJobErrorEvent(ctx, event)
			if err != nil && errors.Cause(err) != slack.ErrUnknownEmail {
				logger.Errorf("http: daemon k8s pod error webhook failed: %+v", err)
				return webhook.NewPostWebhookDaemonK8sJoberrorInternalServerError().WithPayload(unknownError())
			}
			logger.Infof("http: daemon k8s job error webhook: handled")
			return webhook.NewPostWebhookDaemonK8sJoberrorOK().WithPayload(models.EmptyWebhookResponse(struct{}{}))
		})
	}
}
