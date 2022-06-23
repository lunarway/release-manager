package http

import (
	"context"
	"net/http"

	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	policyinternal "github.com/lunarway/release-manager/internal/policy"
	"github.com/lunarway/release-manager/internal/slack"
	opentracing "github.com/opentracing/opentracing-go"
	"gopkg.in/go-playground/webhooks.v5/github"
)

func githubWebhook(payload *payload, flowSvc *flow.Service, policySvc *policyinternal.Service, gitSvc *git.Service, slackClient *slack.Client, githubWebhookSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// copy span from request context but ignore any deadlines on the request context
		ctx := opentracing.ContextWithSpan(context.Background(), opentracing.SpanFromContext(r.Context()))
		logger := log.WithContext(ctx)
		hook, _ := github.New(github.Options.Secret(githubWebhookSecret))
		payload, err := hook.Parse(r, github.PushEvent)
		if err != nil {
			logger.Errorf("http: github webhook: decode request body failed: %v", err)
			invalidBodyError(w)
			return
		}
		switch payload := payload.(type) {
		case github.PushPayload:
			//TODO: Publish payload to fanout exchangge

			w.WriteHeader(http.StatusOK)
			return
		default:
			logger.WithFields("payload", payload).Infof("http: github webhook: payload type '%T': ignored", payload)
			w.WriteHeader(http.StatusOK)
			return
		}
	}
}
