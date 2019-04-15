package flow

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
)

type RollbackResult struct {
	Previous string
	New      string
}

func Rollback(ctx context.Context, opts FlowOptions) (RollbackResult, error) {
	sourceConfigRepoPath, closeSource, err := tempDir("k8s-config-rollback-source")
	if err != nil {
		return RollbackResult{}, err
	}
	defer closeSource()
	destinationConfigRepoPath, closeDestination, err := tempDir("k8s-config-rollback-destination")
	if err != nil {
		return RollbackResult{}, err
	}
	defer closeDestination()
	r, err := git.Clone(ctx, opts.ConfigRepoURL, sourceConfigRepoPath, opts.SSHPrivateKeyPath)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "clone '%s' into '%s'", opts.ConfigRepoURL, sourceConfigRepoPath)
	}

	// locate current release
	currentHash, err := git.LocateServiceRelease(r, opts.Environment, opts.Service)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "locate current release in '%s' at '%s'", opts.ConfigRepoURL, sourceConfigRepoPath)
	}
	log.Debugf("flow: Rollback: current release hash '%v'", currentHash)
	err = git.Checkout(r, currentHash)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "checkout current release hash '%v' in '%s'", currentHash, opts.ConfigRepoURL)
	}
	currentSpec, err := envSpec(sourceConfigRepoPath, opts.ArtifactFileName, opts.Service, opts.Environment)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "get spec of current release hash '%v' in '%s'", currentHash, opts.ConfigRepoURL)
	}

	// locate new release (the previous released artifact for this service)
	newHash, err := git.LocateServiceReleaseRollbackSkip(r, opts.Environment, opts.Service, 1)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "locate previous release in '%s' at '%s'", opts.ConfigRepoURL, sourceConfigRepoPath)
	}
	log.Debugf("flow: Rollback: new release hash '%v'", newHash)
	err = git.Checkout(r, newHash)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "checkout previous release hash '%v' in '%s'", newHash, opts.ConfigRepoURL)
	}
	newSpec, err := envSpec(sourceConfigRepoPath, opts.ArtifactFileName, opts.Service, opts.Environment)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "get spec of previous release hash '%v' in '%s'", newHash, opts.ConfigRepoURL)
	}

	// copy current release artifacts into env
	destinationRepo, err := git.CloneDepth(ctx, opts.ConfigRepoURL, destinationConfigRepoPath, opts.SSHPrivateKeyPath, 1)
	if err != nil {
		return RollbackResult{}, errors.WithMessage(err, fmt.Sprintf("clone destination repo '%s' into '%s'", opts.ConfigRepoURL, destinationConfigRepoPath))
	}

	// release service to env from original release
	sourcePath := releasePath(sourceConfigRepoPath, opts.Service, opts.Environment)
	destinationPath := releasePath(destinationConfigRepoPath, opts.Service, opts.Environment)
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
	artifactSourcePath := path.Join(releasePath(sourceConfigRepoPath, opts.Service, opts.Environment), opts.ArtifactFileName)
	artifactDestinationPath := path.Join(releasePath(destinationConfigRepoPath, opts.Service, opts.Environment), opts.ArtifactFileName)
	log.Infof("flow: ReleaseArtifactID: copy artifact from %s to %s", artifactSourcePath, artifactDestinationPath)
	err = copy.Copy(artifactSourcePath, artifactDestinationPath)
	if err != nil {
		return RollbackResult{}, errors.WithMessage(err, fmt.Sprintf("copy artifact spec from '%s' to '%s'", artifactSourcePath, artifactDestinationPath))
	}

	authorName := newSpec.Application.AuthorName
	authorEmail := newSpec.Application.AuthorEmail
	releaseMessage := git.RollbackCommitMessage(opts.Environment, opts.Service, currentSpec.ID, newSpec.ID)
	err = git.Commit(ctx, destinationRepo, releasePath(".", opts.Service, opts.Environment), authorName, authorEmail, opts.CommitterName, opts.CommitterEmail, releaseMessage, opts.SSHPrivateKeyPath)
	if err != nil {
		return RollbackResult{}, errors.WithMessage(err, fmt.Sprintf("commit changes from path '%s'", destinationPath))
	}
	err = notifyRelease(NotifyReleaseOptions{
		SlackToken:     opts.SlackToken,
		GrafanaApiKey:  opts.GrafanaAPIKey,
		GrafanaBaseUrl: opts.GrafanaUrl,
		Service:        opts.Service,
		Environment:    opts.Environment,
		ArtifactID:     newSpec.ID,
		CommitAuthor:   newSpec.Application.AuthorName,
		CommitMessage:  newSpec.Application.Message,
		CommitSHA:      newSpec.Application.SHA,
		CommitLink:     newSpec.Application.URL,
		Releaser:       opts.CommitterName,
	})
	if err != nil {
		log.Errorf("flow: ReleaseBranch: error notifying release: %v", err)
	}
	log.Infof("flow: Rollback: rollback committed: %s, Author: %s <%s>, Committer: %s <%s>", releaseMessage, authorName, authorEmail, opts.CommitterName, opts.CommitterEmail)
	return RollbackResult{
		Previous: currentSpec.ID,
		New:      newSpec.ID,
	}, nil
}
