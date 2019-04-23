package flow

import (
	"context"
	"regexp"

	"strings"

	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/pkg/errors"
)

func NotifyCommitter(ctx context.Context, configRepoURL, artifactFileName, sshPrivateKeyPath string, event *http.PodNotifyRequest, client *slack.Client, userMappings map[string]string) error {
	sourceConfigRepoPath, close, err := tempDir("k8s-config-notify")
	if err != nil {
		return err
	}
	defer close()
	sourceRepo, err := git.Clone(ctx, configRepoURL, sourceConfigRepoPath, sshPrivateKeyPath)
	if err != nil {
		return errors.WithMessagef(err, "clone '%s' into '%s'", configRepoURL, sourceConfigRepoPath)
	}

	hash, err := git.LocateRelease(sourceRepo, event.ArtifactID)
	if err != nil {
		return errors.WithMessagef(err, "locate release '%s' from '%s'", event.ArtifactID, configRepoURL)
	}

	err = git.Checkout(sourceRepo, hash)
	if err != nil {
		return errors.WithMessagef(err, "checkout release hash '%s' from '%s'", hash, configRepoURL)
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

	sourceSpec, err := envSpec(sourceConfigRepoPath, artifactFileName, service, env)
	if err != nil {
		return errors.WithMessage(err, "locate source spec")
	}

	log.Infof("Commit: %+v", commit)

	if !IsLunarWayEmail(commit.Author.Email) {
		//check UserMappings
		lwEmail, ok := userMappings[commit.Author.Email]
		if !ok {
			log.Errorf("%s is not a Lunar Way email and no mapping exist", commit.Author.Email)
			return errors.Errorf("%s is not a Lunar Way email and no mapping exist", commit.Author.Email)
		}
		commit.Author.Email = lwEmail
	}

	slackUserId, err := client.GetSlackIdByEmail(commit.Author.Email)
	if err != nil {
		return errors.WithMessage(err, "locate slack userId")
	}

	err = client.PostPrivateMessage(slackUserId, env, service, sourceSpec, event)
	if err != nil {
		return errors.WithMessage(err, "post private message")
	}

	return nil
}

func IsLunarWayEmail(email string) bool {
	return strings.Contains(email, "@lunarway.com")
}
