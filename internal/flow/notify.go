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

	// If there's no commits, let's just skips
	if len(event.Commits) == 0 {
		return errors.Errorf("no commits found")
	}

	// Range commit
	for _, commit := range event.Commits {
		log.Infof("Commit Message: %s", commit.Message)
		commitMessage := parseCommitMessage(commit.Message)
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
				span, _ = s.Tracer.FromCtx(ctx, "post flux event processed slack message")
				err := s.Slack.NotifyFluxErrorEvent(ctx, commitMessage.ArtifactID, commitMessage.Environment, email, commitMessage.Service, err.Error, err.Path)
				span.Finish()
				if err != nil {
					return errors.WithMessage(err, "post flux event processed private message")
				}
			}
		}

		// if no errors
		span, _ = s.Tracer.FromCtx(ctx, "post flux event processed slack message")
		err := s.Slack.NotifyFluxEventProcessed(ctx, commitMessage.ArtifactID, commitMessage.Environment, email, commitMessage.Service)
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

func parseCommitMessage(commitMessage string) FluxReleaseMessage {
	r := regexp.MustCompile(`^\[(.*)/(.*)\]\s+release\s+(.*)\s+by\s+(.*)`)
	match := r.FindStringSubmatch(commitMessage)
	return FluxReleaseMessage{
		Environment: match[1],
		Service:     match[2],
		ArtifactID:  match[3],
		GitAuthor:   match[4],
	}
}
