package flow

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/lunarway/release-manager/internal/try"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
)

// NewArtifact should be triggered when a new artifact is ready
func (s *Service) NewArtifact(ctx context.Context, service, artifactID string) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.NewArtifact")
	defer span.Finish()

	err := s.PublishNewArtifact(ctx, NewArtifactEvent{
		Service:    service,
		ArtifactID: artifactID,
	})
	if err != nil {
		return err
	}

	return nil
}

type NewArtifactEvent struct {
	Service    string `json:"service,omitempty"`
	ArtifactID string `json:"artifactId,omitempty"`
}

func (NewArtifactEvent) Type() string {
	return "newArtifact"
}

func (p NewArtifactEvent) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *NewArtifactEvent) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}

// ExecNewArtifact is handling behavior of release manager when new artifacts are generated and ready
func (s *Service) ExecNewArtifact(ctx context.Context, e NewArtifactEvent) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.ExecNewArtifact")
	defer span.Finish()

	logger := log.WithContext(ctx)

	artifactSpec, err := s.Storage.ArtifactSpecification(ctx, e.Service, e.ArtifactID)

	logger = logger.WithFields("branch", artifactSpec.Application.Branch, "service", artifactSpec.Service, "commit", artifactSpec.Application.SHA)
	// lookup policies for branch
	autoReleases, err := s.Policy.GetAutoReleases(ctx, artifactSpec.Service, artifactSpec.Application.Branch)
	if err != nil {
		logger.Errorf("flow: exec new artifact: service '%s' branch '%s': get auto release policies failed: %v", artifactSpec.Service, artifactSpec.Application.Branch, err)
		err := s.Slack.NotifySlackPolicyFailed(ctx, artifactSpec.Application.AuthorEmail, ":rocket: Release Manager :no_entry:", fmt.Sprintf("Auto release policy failed for service %s and %s", artifactSpec.Service, artifactSpec.Application.Branch))
		if err != nil {
			logger.Errorf("flow: exec new artifact: get auto-release policies: error notifying slack: %v", err)
		}
		return err
	}
	logger.Infof("flow: exec new artifact: service '%s' branch '%s': found %d release policies", artifactSpec.Service, artifactSpec.Application.Branch, len(autoReleases))
	var errs error
	for _, autoRelease := range autoReleases {
		releaseID, err := s.ReleaseBranch(ctx, Actor{
			Name:  artifactSpec.Application.AuthorName,
			Email: artifactSpec.Application.AuthorEmail,
		}, autoRelease.Environment, artifactSpec.Service, autoRelease.Branch)
		if err != nil {
			if errorCause(err) != git.ErrNothingToCommit {
				errs = multierr.Append(errs, err)
				err := s.Slack.NotifySlackPolicyFailed(ctx, artifactSpec.Application.AuthorEmail, ":rocket: Release Manager :no_entry:", fmt.Sprintf("Service %s was not released into %s from branch %s.\nYou can deploy manually using `hamctl`:\nhamctl release --service %[1]s --branch %[3]s --env %[2]s", artifactSpec.Service, autoRelease.Environment, autoRelease.Branch))
				if err != nil {
					logger.Errorf("flow: exec new artifact: auto-release failed: error notifying slack: %v", err)
				}
				continue
			}
			logger.Infof("flow: exec new artifact: service '%s': auto-release from policy '%s' to '%s': %v", artifactSpec.Service, autoRelease.ID, autoRelease.Environment, err)
			continue
		}
		//TODO: Parse and switch to signoff user
		err = s.Slack.NotifySlackPolicySucceeded(ctx, artifactSpec.Application.AuthorEmail, ":rocket: Release Manager :white_check_mark:", fmt.Sprintf("Service *%s* will be auto released to *%s*\nArtifact: <%s|*%s*>", artifactSpec.Service, autoRelease.Environment, artifactSpec.Application.URL, releaseID))
		if err != nil {
			if errors.Cause(err) != slack.ErrUnknownEmail {
				logger.Errorf("flow: exec new artifact: auto-release succeeded: error notifying slack: %v", err)
			}
		}
		logger.Infof("flow: exec new artifact: service '%s': auto-release from policy '%s' of %s to %s", artifactSpec.Service, autoRelease.ID, releaseID, autoRelease.Environment)
	}
	if errs != nil {
		logger.Errorf("flow: exec new artifact: service '%s' branch '%s': auto-release failed with one or more errors: %v", artifactSpec.Service, artifactSpec.Application.Branch, errs)
		return errs
	}

	if err != nil {
		return err
	}
	return nil
}

func errorCause(err error) error {
	// get cause before and after multierr unwrap to handle wrapped multierrs and
	// multierrs with wrapped errors
	errs := multierr.Errors(errors.Cause(err))
	if len(errs) == 0 {
		return nil
	}
	for i := len(errs) - 1; i >= 0; i-- {
		err := errs[i]
		if err != try.ErrTooManyRetries {
			return errors.Cause(err)
		}
	}
	return err
}
