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
	sourceConfigRepoPath, close, err := tempDir("k8s-config-notify")
	if err != nil {
		return err
	}
	defer close()
	sourceRepo, err := git.Clone(ctx, s.ConfigRepoURL, sourceConfigRepoPath, s.SSHPrivateKeyPath)
	if err != nil {
		return errors.WithMessagef(err, "clone '%s' into '%s'", s.ConfigRepoURL, sourceConfigRepoPath)
	}

	hash, err := git.LocateRelease(sourceRepo, event.ArtifactID)
	if err != nil {
		return errors.WithMessagef(err, "locate release '%s' from '%s'", event.ArtifactID, s.ConfigRepoURL)
	}

	err = git.Checkout(sourceRepo, hash)
	if err != nil {
		return errors.WithMessagef(err, "checkout release hash '%s' from '%s'", hash, s.ConfigRepoURL)
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
	env := matches[1]
	service := matches[2]

	sourceSpec, err := envSpec(sourceConfigRepoPath, s.ArtifactFileName, service, env)
	if err != nil {
		return errors.WithMessage(err, "locate source spec")
	}

	log.Infof("Commit: %+v", commit)

	if !IsLunarWayEmail(commit.Author.Email) {
		//check UserMappings
		lwEmail, ok := s.UserMappings[commit.Author.Email]
		if !ok {
			log.Errorf("%s is not a Lunar Way email and no mapping exist", commit.Author.Email)
			return errors.Errorf("%s is not a Lunar Way email and no mapping exist", commit.Author.Email)
		}
		commit.Author.Email = lwEmail
	}

	slackUserId, err := s.Slack.GetSlackIdByEmail(commit.Author.Email)
	if err != nil {
		return errors.WithMessage(err, "locate slack userId")
	}

	err = s.Slack.PostPrivateMessage(slackUserId, env, service, sourceSpec, event)
	if err != nil {
		return errors.WithMessage(err, "post private message")
	}

	return nil
}

func IsLunarWayEmail(email string) bool {
	return strings.Contains(email, "@lunarway.com")
}
