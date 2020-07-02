package flow

import (
	"context"

	"github.com/lunarway/release-manager/internal/commitinfo"
	"github.com/lunarway/release-manager/internal/flux"
	"github.com/lunarway/release-manager/internal/http"

	"github.com/pkg/errors"
)

func (s *Service) NotifyK8SDeployEvent(ctx context.Context, event *http.ReleaseEvent) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.NotifyK8SDeployment")
	defer span.Finish()
	span, _ = s.Tracer.FromCtx(ctx, "post k8s deploy slack message")
	err := s.Slack.NotifyK8SDeployEvent(ctx, event)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "post k8s deploy slack message")
	}
	return nil
}

func (s *Service) NotifyK8SPodErrorEvent(ctx context.Context, event *http.PodErrorEvent) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.NotifyK8SPodErrorEvent")
	defer span.Finish()
	span, _ = s.Tracer.FromCtx(ctx, "post k8s NotifyK8SPodErrorEvent slack message")
	err := s.Slack.NotifyK8SPodErrorEvent(ctx, event)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "post k8s NotifyK8SPodErrorEvent slack message")
	}
	return nil
}

func (s *Service) NotifyFluxEvent(ctx context.Context, event *http.FluxNotifyRequest) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.NotifyFluxEvent")
	defer span.Finish()

	fluxCommits := flux.GetCommits(event.FluxEvent.Metadata)
	fluxErrors := flux.GetErrors(event.FluxEvent.Metadata)

	// If there's no commits, let's just skip
	if len(fluxCommits) == 0 {
		return nil
	}

	for _, commit := range fluxCommits {
		commitMessage, err := commitinfo.ParseCommitInfo(commit.Message)
		if err != nil {
			return errors.WithMessagef(err, "parse commit message '%s'", commit.Message)
		}

		// event contains errors, extract and post specific error message
		if len(fluxErrors) > 0 {
			for _, err := range fluxErrors {
				span, ctx := s.Tracer.FromCtx(ctx, "post flux error event slack message")
				err := s.Slack.NotifyFluxErrorEvent(ctx, commitMessage.ArtifactID, commitMessage.Environment, commitMessage.ReleasedBy.Email, commitMessage.Service, err.Error, err.Path)
				span.Finish()
				if err != nil {
					return errors.WithMessage(err, "post flux event processed private message")
				}
			}
			return nil
		}

		// if no errors
		span, ctx = s.Tracer.FromCtx(ctx, "post flux event processed slack message")
		err = s.Slack.NotifyFluxEventProcessed(ctx, commitMessage.ArtifactID, commitMessage.Environment, commitMessage.ReleasedBy.Email, commitMessage.Service)
		span.Finish()
		if err != nil {
			return errors.WithMessagef(err, "post flux event processed private message; artifact: %s, env: %s, email: %s, service: %s", commitMessage.ArtifactID, commitMessage.Environment, commitMessage.ReleasedBy.Email, commitMessage.Service)
		}
	}
	return nil
}
