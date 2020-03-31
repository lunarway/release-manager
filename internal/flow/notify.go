package flow

import (
	"context"
	"regexp"

	"strings"

	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

func (s *Service) NotifyCommitter(ctx context.Context, event *http.PodNotifyRequest) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.NotifyCommitter")
	defer span.Finish()
	email := event.AuthorEmail
	if !strings.Contains(email, "@lunar.app") {
		//check UserMappings
		lwEmail, ok := s.UserMappings[email]
		if !ok {
			log.WithContext(ctx).Errorf("%s is not a Lunar Way email and no mapping exist", email)
			return errors.Errorf("%s is not a Lunar Way email and no mapping exist", email)
		}
		email = lwEmail
	}
	span, _ = s.Tracer.FromCtx(ctx, "post private slack message")
	err := s.Slack.PostPrivateMessage(ctx, email, event)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "post private message")
	}

	return nil
}

func (s *Service) NotifyFluxEvent(ctx context.Context, event *http.FluxNotifyRequest) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.NotifyFluxEvent")
	defer span.Finish()

	// If there's no commits, let's just skip
	if len(event.Commits) == 0 {
		return nil
	}

	for _, commit := range event.Commits {
		commitMessage, err := parseCommitMessage(commit.Message)
		if err != nil {
			return err
		}
		email := commitMessage.GitAuthor
		if !strings.Contains(email, "@lunar.app") {
			//check UserMappings
			lwEmail, ok := s.UserMappings[email]
			if !ok {
				log.WithContext(ctx).Errorf("%s is not a Lunar email and no mapping exist", email)
				return errors.Errorf("%s is not a Lunar email and no mapping exist", email)
			}
			email = lwEmail
		}
		// event contains errors, extract and post specific error message
		if len(event.Errors) > 0 {
			for _, err := range event.Errors {
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
