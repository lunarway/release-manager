package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/lunarway/release-manager/internal/commitinfo"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	policyinternal "github.com/lunarway/release-manager/internal/policy"
	"github.com/lunarway/release-manager/internal/slack"
	opentracing "github.com/opentracing/opentracing-go"
	"gopkg.in/go-playground/webhooks.v5/github"
)

func githubWebhook(payload *payload, flowSvc *flow.Service, policySvc *policyinternal.Service, gitSvc *git.Service, slackClient *slack.Client, githubWebhookSecret string) http.HandlerFunc {
	commitMessageExtractorFunc := commitinfo.ExtractInfoFromCommit()
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
			if !isBranchPush(payload.Ref) {
				logger.Infof("http: github webhook: ref '%s' is not a branch push", payload.Ref)
				w.WriteHeader(http.StatusOK)
				return
			}
			err := gitSvc.SyncMaster(ctx)
			if err != nil {
				logger.Errorf("http: github webhook: failed to sync master: %v", err)
				w.WriteHeader(http.StatusOK)
				return
			}
			commitInfo, err := commitMessageExtractorFunc(payload.HeadCommit.Message)
			if err != nil {
				logger.Infof("http: github webhook: extract author details from commit failed: message '%s'", payload.HeadCommit.Message)
				w.WriteHeader(http.StatusOK)
				return
			}

			err = flowSvc.NewArtifact(ctx, commitInfo.Service, commitInfo.ArtifactID)
			if err != nil {
				logger.Infof("http: github webhook: service '%s': could not publish new artifact event for %s: %v", commitInfo.Service, commitInfo.ArtifactID, err)
				unknownError(w)
				return
			}
			logger.Infof("http: github webhook: handled successfully: service '%s' commit '%s'", commitInfo.Service, payload.HeadCommit.ID)
			w.WriteHeader(http.StatusOK)
			return
		default:
			logger.WithFields("payload", payload).Infof("http: github webhook: payload type '%T': ignored", payload)
			w.WriteHeader(http.StatusOK)
			return
		}
	}
}

func isBranchPush(ref string) bool {
	return strings.HasPrefix(ref, "refs/heads/")
}
