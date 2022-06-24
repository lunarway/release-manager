package http

import (
	"context"
	"net/http"

	"github.com/lunarway/release-manager/internal/log"
	opentracing "github.com/opentracing/opentracing-go"
	"gopkg.in/go-playground/webhooks.v5/github"
)

func githubWebhook(githubWebhookSecret string, publisher func(ctx context.Context, payload github.PushPayload) error) http.HandlerFunc {
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
			err = publisher(ctx, payload)
			if err != nil {
				logger.Errorf("http: github webhook: publish payload failed: %v", err)
				unknownError(w)
			}

			w.WriteHeader(http.StatusOK)
			return
		default:
			logger.WithFields("payload", payload).Infof("http: github webhook: payload type '%T': ignored", payload)
			w.WriteHeader(http.StatusOK)
			return
		}
	}
}
