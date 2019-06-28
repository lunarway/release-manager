package flow

import (
	"context"
	"fmt"
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
	span, ctx := s.Tracer.FromCtx(ctx, "flow.Rollback")
	defer span.Finish()
	var result RollbackResult
	err := s.retry(ctx, func(ctx context.Context, attempt int) (bool, error) {
		sourceConfigRepoPath, closeSource, err := git.TempDir(ctx, s.Tracer, "k8s-config-rollback-source")
		if err != nil {
			return true, err
		}
		defer closeSource(ctx)
		destinationConfigRepoPath, closeDestination, err := git.TempDir(ctx, s.Tracer, "k8s-config-rollback-destination")
		if err != nil {
			return true, err
		}
		defer closeDestination(ctx)
		r, err := s.Git.Clone(ctx, sourceConfigRepoPath)
		if err != nil {
			return true, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
		}

		// locate current release
		currentHash, err := s.Git.LocateServiceRelease(ctx, r, environment, service)
		if err != nil {
			return true, errors.WithMessagef(err, "locate current release at '%s'", sourceConfigRepoPath)
		}
		log.Debugf("flow: Rollback: current release hash '%v'", currentHash)
		err = s.Git.Checkout(ctx, sourceConfigRepoPath, currentHash)
		if err != nil {
			return true, errors.WithMessagef(err, "checkout current release hash '%v'", currentHash)
		}
		// default to environment name for the namespace if none is specified
		if namespace == "" {
			namespace = environment
		}
		log.Infof("flow: Rollback: using namespace '%s'", namespace)

		currentSpec, err := envSpec(sourceConfigRepoPath, s.ArtifactFileName, service, environment, namespace)
		if err != nil {
			return true, errors.WithMessagef(err, "get spec of current release hash '%v'", currentHash)
		}

		// if artifact has no namespace we only allow using the environment as
		// namespace.
		if currentSpec.Namespace == "" && namespace != environment {
			return true, errors.WithMessagef(ErrNamespaceNotAllowedByArtifact, "namespace '%s'", namespace)
		}

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
		newHash, err := s.Git.LocateServiceReleaseRollbackSkip(ctx, r, environment, service, 1)
		if err != nil {
			return true, errors.WithMessagef(err, "locate previous release at '%s'", sourceConfigRepoPath)
		}
		log.Debugf("flow: Rollback: new release hash '%v'", newHash)
		err = s.Git.Checkout(ctx, sourceConfigRepoPath, newHash)
		if err != nil {
			return true, errors.WithMessagef(err, "checkout previous release hash '%v'", newHash)
		}
		newSpec, err := envSpec(sourceConfigRepoPath, s.ArtifactFileName, service, environment, namespace)
		if err != nil {
			return true, errors.WithMessagef(err, "get spec of previous release hash '%v'", newHash)
		}

		// copy current release artifacts into env
		destinationRepo, err := s.Git.Clone(ctx, destinationConfigRepoPath)
		if err != nil {
			return true, errors.WithMessagef(err, "clone destination repo into '%s'", destinationConfigRepoPath)
		}

		// release service to env from original release
		sourcePath := releasePath(sourceConfigRepoPath, service, environment, namespace)
		destinationPath := releasePath(destinationConfigRepoPath, service, environment, namespace)
		log.Infof("flow: ReleaseArtifactID: copy resources from %s to %s", sourcePath, destinationPath)

		err = s.cleanCopy(ctx, sourcePath, destinationPath)
		if err != nil {
			return true, errors.WithMessagef(err, "copy resources from '%s' to '%s'", sourcePath, destinationPath)
		}
		// copy artifact spec
		artifactSourcePath := path.Join(releasePath(sourceConfigRepoPath, service, environment, namespace), s.ArtifactFileName)
		artifactDestinationPath := path.Join(releasePath(destinationConfigRepoPath, service, environment, namespace), s.ArtifactFileName)
		log.Infof("flow: ReleaseArtifactID: copy artifact from %s to %s", artifactSourcePath, artifactDestinationPath)
		err = copy.Copy(artifactSourcePath, artifactDestinationPath)
		if err != nil {
			return true, errors.WithMessage(err, fmt.Sprintf("copy artifact spec from '%s' to '%s'", artifactSourcePath, artifactDestinationPath))
		}

		authorName := newSpec.Application.AuthorName
		authorEmail := newSpec.Application.AuthorEmail
		releaseMessage := git.RollbackCommitMessage(environment, service, currentSpec.ID, newSpec.ID)
		err = s.Git.Commit(ctx, destinationRepo, destinationConfigRepoPath, releasePath(".", service, environment, namespace), authorName, authorEmail, actor.Name, actor.Email, releaseMessage)
		if err != nil {
			if err == git.ErrNothingToCommit {
				return true, errors.WithMessage(err, fmt.Sprintf("commit changes from path '%s'", destinationPath))
			}
			// we can see races here where other changes are committed to the master repo
			// after we cloned. Because of this we retry on any error.
			return false, errors.WithMessage(err, fmt.Sprintf("commit changes from path '%s'", destinationPath))
		}
		err = s.notifyRelease(ctx, NotifyReleaseOptions{
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
		return true, nil
	})
	if err != nil {
		return RollbackResult{}, err
	}
	return result, nil
}
