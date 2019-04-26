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
	Previous             string
	New                  string
	OverwritingNamespace string
}

func (s *Service) Rollback(ctx context.Context, actor Actor, environment, namespace, service string) (RollbackResult, error) {
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
	r, err := git.Clone(ctx, s.ConfigRepoURL, sourceConfigRepoPath, s.SSHPrivateKeyPath)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "clone '%s' into '%s'", s.ConfigRepoURL, sourceConfigRepoPath)
	}

	// locate current release
	currentHash, err := git.LocateServiceRelease(r, environment, service)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "locate current release in '%s' at '%s'", s.ConfigRepoURL, sourceConfigRepoPath)
	}
	log.Debugf("flow: Rollback: current release hash '%v'", currentHash)
	err = git.Checkout(r, currentHash)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "checkout current release hash '%v' in '%s'", currentHash, s.ConfigRepoURL)
	}
	// default to environment name for the namespace if none is specified
	if namespace == "" {
		namespace = environment
	}
	log.Infof("flow: Rollback: using namespace '%s'", namespace)

	currentSpec, err := envSpec(sourceConfigRepoPath, s.ArtifactFileName, service, environment, namespace)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "get spec of current release hash '%v' in '%s'", currentHash, s.ConfigRepoURL)
	}

	// if artifact has no namespace we only allow using the environment as
	// namespace.
	if currentSpec.Namespace == "" && namespace != environment {
		return RollbackResult{}, errors.WithMessagef(ErrNamespaceNotAllowedByArtifact, "namespace '%s'", namespace)
	}

	var result RollbackResult
	// a developer can mistakenly specify the wrong namespace when promoting and
	// we will not be able to detect it before this point.
	// It only affects "dev" promotes as we read from the artifacts here where we
	// can find the artifact without taking the namespace into account.
	if currentSpec.Namespace != "" && namespace != currentSpec.Namespace {
		log.Infof("flow: Rollback: overwriting namespace '%s' to '%s'", namespace, currentSpec.Namespace)
		namespace = currentSpec.Namespace
		result.OverwritingNamespace = currentSpec.Namespace
	}
	// locate new release (the previous released artifact for this service)
	newHash, err := git.LocateServiceReleaseRollbackSkip(r, environment, service, 1)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "locate previous release in '%s' at '%s'", s.ConfigRepoURL, sourceConfigRepoPath)
	}
	log.Debugf("flow: Rollback: new release hash '%v'", newHash)
	err = git.Checkout(r, newHash)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "checkout previous release hash '%v' in '%s'", newHash, s.ConfigRepoURL)
	}
	newSpec, err := envSpec(sourceConfigRepoPath, s.ArtifactFileName, service, environment, namespace)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "get spec of previous release hash '%v' in '%s'", newHash, s.ConfigRepoURL)
	}

	// copy current release artifacts into env
	destinationRepo, err := git.CloneDepth(ctx, s.ConfigRepoURL, destinationConfigRepoPath, s.SSHPrivateKeyPath, 1)
	if err != nil {
		return RollbackResult{}, errors.WithMessage(err, fmt.Sprintf("clone destination repo '%s' into '%s'", s.ConfigRepoURL, destinationConfigRepoPath))
	}

	// release service to env from original release
	sourcePath := releasePath(sourceConfigRepoPath, service, environment, namespace)
	destinationPath := releasePath(destinationConfigRepoPath, service, environment, namespace)
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
	artifactSourcePath := path.Join(releasePath(sourceConfigRepoPath, service, environment, namespace), s.ArtifactFileName)
	artifactDestinationPath := path.Join(releasePath(destinationConfigRepoPath, service, environment, namespace), s.ArtifactFileName)
	log.Infof("flow: ReleaseArtifactID: copy artifact from %s to %s", artifactSourcePath, artifactDestinationPath)
	err = copy.Copy(artifactSourcePath, artifactDestinationPath)
	if err != nil {
		return RollbackResult{}, errors.WithMessage(err, fmt.Sprintf("copy artifact spec from '%s' to '%s'", artifactSourcePath, artifactDestinationPath))
	}

	authorName := newSpec.Application.AuthorName
	authorEmail := newSpec.Application.AuthorEmail
	releaseMessage := git.RollbackCommitMessage(environment, service, currentSpec.ID, newSpec.ID)
	err = git.Commit(ctx, destinationRepo, releasePath(".", service, environment, namespace), authorName, authorEmail, actor.Name, actor.Email, releaseMessage, s.SSHPrivateKeyPath)
	if err != nil {
		return RollbackResult{}, errors.WithMessage(err, fmt.Sprintf("commit changes from path '%s'", destinationPath))
	}
	err = s.notifyRelease(NotifyReleaseOptions{
		Service:       service,
		Environment:   environment,
		Namespace:     namespace,
		ArtifactID:    newSpec.ID,
		CommitAuthor:  newSpec.Application.AuthorName,
		CommitMessage: newSpec.Application.Message,
		CommitSHA:     newSpec.Application.SHA,
		CommitLink:    newSpec.Application.URL,
		Releaser:      actor.Name,
	})
	if err != nil {
		log.Errorf("flow: ReleaseBranch: error notifying release: %v", err)
	}
	log.Infof("flow: Rollback: rollback committed: %s, Author: %s <%s>, Committer: %s <%s>", releaseMessage, authorName, authorEmail, actor.Name, actor.Email)
	result.Previous = currentSpec.ID
	result.New = newSpec.ID
	return result, nil
}
