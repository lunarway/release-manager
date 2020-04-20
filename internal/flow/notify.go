package flow

import (
	"context"
	"regexp"

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
		commitMessage, err := parseCommitMessage(commit.Message)
		if err != nil {
			return errors.WithMessagef(err, "parse commit message '%s'", commit.Message)
		}
		email := commitMessage.GitAuthor

		// event contains errors, extract and post specific error message
		if len(fluxErrors) > 0 {
			for _, err := range fluxErrors {
				span, ctx := s.Tracer.FromCtx(ctx, "post flux error event slack message")
				err := s.Slack.NotifyFluxErrorEvent(ctx, commitMessage.ArtifactID, commitMessage.Environment, email, commitMessage.Service, err.Error, err.Path)
				span.Finish()
				if err != nil {
					return errors.WithMessage(err, "post flux event processed private message")
				}
			}
			return nil
		}

		// if no errors
		span, ctx = s.Tracer.FromCtx(ctx, "post flux event processed slack message")
		err = s.Slack.NotifyFluxEventProcessed(ctx, commitMessage.ArtifactID, commitMessage.Environment, email, commitMessage.Service)
		span.Finish()
		if err != nil {
			return errors.WithMessage(err, "post flux event processed private message")
		}
	}
	return nil
}

type FluxReleaseMessage struct {
	Environment string
	Service     string
	ArtifactID  string
	GitAuthor   string
}

func parseCommitMessage(commitMessage string) (FluxReleaseMessage, error) {
	pattern := `^\[(?P<env>.*)/(?P<service>.*)\]\s+release\s+(?P<artifact>.*)\s+by\s+(?P<author>.*)`
	r, err := regexp.Compile(pattern)
	if err != nil {
		return FluxReleaseMessage{}, errors.WithMessage(err, "regex didn't match")
	}
	matches := r.FindStringSubmatch(commitMessage)
	if len(matches) < 1 {
		return FluxReleaseMessage{}, errors.New("not enough matches")
	}

	return FluxReleaseMessage{
		Environment: matches[1],
		Service:     matches[2],
		ArtifactID:  matches[3],
		GitAuthor:   matches[4],
	}, nil
}
