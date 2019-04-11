package flow

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
)

type RollbackResult struct {
	Previous string
	New      string
}

func Rollback(ctx context.Context, configRepoURL, artifactFileName, service, env, committerName, committerEmail, sshPrivateKeyPath, slackToken string) (RollbackResult, error) {
	r, err := git.Clone(ctx, configRepoURL, sourceConfigRepoPath, sshPrivateKeyPath)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "clone '%s' into '%s'", configRepoURL, sourceConfigRepoPath)
	}

	// locate current release
	currentHash, err := git.LocateServiceRelease(r, env, service)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "locate current release in '%s' at '%s'", configRepoURL, sourceConfigRepoPath)
	}
	log.Debugf("flow: Rollback: current release hash '%v'", currentHash)
	err = git.Checkout(r, currentHash)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "checkout current release hash '%v' in '%s'", currentHash, configRepoURL)
	}
	currentSpec, err := envSpec(sourceConfigRepoPath, artifactFileName, service, env)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "get spec of current release hash '%v' in '%s'", currentHash, configRepoURL)
	}

	// locate new release (the previous released artifact for this service)
	newHash, err := git.LocateServiceReleaseRollbackSkip(r, env, service, 1)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "locate previous release in '%s' at '%s'", configRepoURL, sourceConfigRepoPath)
	}
	log.Debugf("flow: Rollback: new release hash '%v'", newHash)
	err = git.Checkout(r, newHash)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "checkout previous release hash '%v' in '%s'", newHash, configRepoURL)
	}
	newSpec, err := envSpec(sourceConfigRepoPath, artifactFileName, service, env)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "get spec of previous release hash '%v' in '%s'", newHash, configRepoURL)
	}

	// copy current release artifacts into env
	destinationRepo, err := git.CloneDepth(ctx, configRepoURL, destinationConfigRepoPath, sshPrivateKeyPath, 1)
	if err != nil {
		return RollbackResult{}, errors.WithMessage(err, fmt.Sprintf("clone destination repo '%s' into '%s'", configRepoURL, destinationConfigRepoPath))
	}

	// release service to env from original release
	sourcePath := releasePath(sourceConfigRepoPath, service, env)
	destinationPath := releasePath(destinationConfigRepoPath, service, env)
	log.Infof("flow: ReleaseArtifactID: copy resources from %s to %s", sourcePath, destinationPath)

	// empty existing resources in destination
	err = os.RemoveAll(destinationPath)
	if err != nil {
		return RollbackResult{}, errors.WithMessage(err, fmt.Sprintf("remove destination path '%s'", destinationPath))
	}
	err = os.MkdirAll(destinationPath, os.ModePerm)
	if err != nil {
		return RollbackResult{}, errors.WithMessage(err, fmt.Sprintf("create destination dir '%s'", destinationPath))
	}
	// copy previous env. files into destination
	err = copy.Copy(sourcePath, destinationPath)
	if err != nil {
		return RollbackResult{}, errors.WithMessage(err, fmt.Sprintf("copy resources from '%s' to '%s'", sourcePath, destinationPath))
	}
	// copy artifact spec
	artifactSourcePath := path.Join(releasePath(sourceConfigRepoPath, service, env), artifactFileName)
	artifactDestinationPath := path.Join(releasePath(destinationConfigRepoPath, service, env), artifactFileName)
	log.Infof("flow: ReleaseArtifactID: copy artifact from %s to %s", artifactSourcePath, artifactDestinationPath)
	err = copy.Copy(artifactSourcePath, artifactDestinationPath)
	if err != nil {
		return RollbackResult{}, errors.WithMessage(err, fmt.Sprintf("copy artifact spec from '%s' to '%s'", artifactSourcePath, artifactDestinationPath))
	}

	authorName := newSpec.Application.AuthorName
	authorEmail := newSpec.Application.AuthorEmail
	releaseMessage := git.RollbackCommitMessage(env, service, currentSpec.ID, newSpec.ID)
	err = git.Commit(ctx, destinationRepo, releasePath(".", service, env), authorName, authorEmail, committerName, committerEmail, releaseMessage, sshPrivateKeyPath)
	if err != nil {
		return RollbackResult{}, errors.WithMessage(err, fmt.Sprintf("commit changes from path '%s'", destinationPath))
	}
	err = notifyRelease(slack.ReleaseOptions{
		SlackToken:    slackToken,
		Service:       service,
		Environment:   env,
		ArtifactID:    newSpec.ID,
		CommitAuthor:  newSpec.Application.AuthorName,
		CommitMessage: newSpec.Application.Message,
		CommitSHA:     newSpec.Application.SHA,
		CommitLink:    newSpec.Application.URL,
		Releaser:      committerName,
	})
	if err != nil {
		log.Errorf("flow: ReleaseBranch: error notifying release: %v", err)
	}
	log.Infof("flow: Rollback: rollback committed: %s, Author: %s <%s>, Committer: %s <%s>", releaseMessage, authorName, authorEmail, committerName, committerEmail)
	return RollbackResult{
		Previous: currentSpec.ID,
		New:      newSpec.ID,
	}, nil
}
