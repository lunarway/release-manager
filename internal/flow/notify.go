package flow

import (
	"context"
	"fmt"
	"regexp"

	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/pkg/errors"
)

func NotifyCommitter(ctx context.Context, configRepoURL, artifactFileName, sshPrivateKeyPath string, event *http.PodNotifyRequest, client *slack.Client) error {
	sourceRepo, err := git.Clone(ctx, configRepoURL, sourceConfigRepoPath, sshPrivateKeyPath)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("clone '%s' into '%s'", configRepoURL, sourceConfigRepoPath))
	}

	hash, err := git.LocateRelease(sourceRepo, event.ArtifactID)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("locate release '%s' from '%s'", event.ArtifactID, configRepoURL))
	}

	err = git.Checkout(sourceRepo, hash)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("checkout release hash '%s' from '%s'", hash, configRepoURL))
	}

	commit, err := sourceRepo.CommitObject(hash)
	if err != nil {
		return errors.WithMessage(err, "locate commit object")
	}

	rgx := regexp.MustCompile(`\[(.*?)\/(.*?)\]`)
	matches := rgx.FindStringSubmatch(commit.Message)
	if len(matches) < 2 {
		return errors.WithMessage(err, fmt.Sprintf("locate service from commit message: '%s'", commit.Message))
	}
	env := matches[1]
	service := matches[2]

	sourceSpec, err := sourceSpec(sourceConfigRepoPath, artifactFileName, service, env)
	if err != nil {
		return errors.WithMessage(err, "locate source spec")
	}

	log.Infof("Commit: %+v", commit)

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
