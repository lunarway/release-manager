package http

import (
	"context"
	"strings"

	"github.com/go-openapi/runtime/middleware"
	"github.com/lunarway/release-manager/generated/http/restapi/operations"
	"github.com/lunarway/release-manager/generated/http/restapi/operations/webhook"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	opentracing "github.com/opentracing/opentracing-go"
	"gopkg.in/go-playground/webhooks.v5/github"
)

func GithubWebhookHandler(gitSvc *git.Service, githubWebhookSecret string) HandlerFactory {
	return func(api *operations.ReleaseManagerServerAPIAPI) {
		api.WebhookPostWebhookGithubHandler = webhook.PostWebhookGithubHandlerFunc(func(params webhook.PostWebhookGithubParams) middleware.Responder {
			// copy span from request context but ignore any deadlines on the request context
			ctx := opentracing.ContextWithSpan(context.Background(), opentracing.SpanFromContext(params.HTTPRequest.Context()))
			logger := log.WithContext(ctx)

			hook, _ := github.New(github.Options.Secret(githubWebhookSecret))

			payload, err := hook.Parse(params.HTTPRequest, github.PushEvent)
			if err != nil {
				logger.Errorf("http: github webhook: decode request body failed: %v", err)
				return webhook.NewPostWebhookGithubBadRequest().WithPayload(badRequest("invalid body"))
			}

			switch payload := payload.(type) {
			case github.PushPayload:
				if !isBranchPush(payload.Ref) {
					logger.Infof("http: github webhook: ref '%s' is not a branch push", payload.Ref)
					return webhook.NewPostWebhookGithubOK()
				}
				err := gitSvc.SyncMaster(ctx)
				if err != nil {
					logger.Errorf("http: github webhook: failed to sync master: %v", err)
					return webhook.NewPostWebhookGithubOK()
				}
				return webhook.NewPostWebhookGithubOK()
			default:
				logger.WithFields("payload", payload).Infof("http: github webhook: payload type '%T': ignored", payload)
				return webhook.NewPostWebhookGithubOK()
			}
		})
	}
}

func isBranchPush(ref string) bool {
	return strings.HasPrefix(ref, "refs/heads/")
}
