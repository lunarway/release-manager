package flow

import (
	"context"
	"regexp"

	"strings"

	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

func (s *Service) NotifyCommitter(ctx context.Context, event *http.PodNotifyRequest) error {
	span, ctx := s.span(ctx, "flow.NotifyCommitter")
	defer span.Finish()
	sourceConfigRepoPath, close, err := git.TempDir(ctx, s.Tracer, "k8s-config-notify")
	if err != nil {
		return err
	}
	defer close(ctx)
	sourceRepo, err := s.Git.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		return errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
	}

	hash, err := s.Git.LocateEnvRelease(ctx, sourceRepo, event.Environment, event.ArtifactID)
	if err != nil {
		return errors.WithMessagef(err, "locate release '%s'", event.ArtifactID)
	}
	log.Infof("internal/flow: NotifyCommitter: located release of '%s' on hash '%s'", event.ArtifactID, hash)

	err = s.Git.Checkout(ctx, sourceRepo, hash)
	if err != nil {
		return errors.WithMessagef(err, "checkout release hash '%s'", hash)
	}

	commit, err := sourceRepo.CommitObject(hash)
	if err != nil {
		return errors.WithMessage(err, "locate commit object")
	}

	rgx := regexp.MustCompile(`\[(.*?)\/(.*?)\]`)
	matches := rgx.FindStringSubmatch(commit.Message)
	if len(matches) < 2 {
		return errors.WithMessagef(err, "locate service from commit message: '%s'", commit.Message)
	}
	// Use environment received from release-daemon
	env := event.Environment
	service := matches[2]
	namespace := event.Namespace

	log.Infof("internal/flow: NotifyCommitter: read spec from: env '%s' namespace '%s' service '%s'", env, namespace, service)
	sourceSpec, err := envSpec(sourceConfigRepoPath, s.ArtifactFileName, service, env, namespace)
	if err != nil {
		return errors.WithMessage(err, "locate source spec")
	}

	log.Infof("Commit: %+v", commit)
	email := commit.Committer.Email
	if email == "" {
		email = commit.Author.Email
	}
	if !strings.Contains(email, "@lunarway.com") {
		//check UserMappings
		lwEmail, ok := s.UserMappings[email]
		if !ok {
			log.Errorf("%s is not a Lunar Way email and no mapping exist", email)
			return errors.Errorf("%s is not a Lunar Way email and no mapping exist", email)
		}
		email = lwEmail
	}
	span, _ = s.span(ctx, "post private slack message")
	err = s.Slack.PostPrivateMessage(email, env, service, sourceSpec, event)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "post private message")
	}

	return nil
}
