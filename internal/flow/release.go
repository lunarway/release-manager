package flow

import (
	"context"
	"fmt"
	"path"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
)

// ReleaseBranch releases the latest artifact from a branch of a specific
// service to environment env.
//
// Flow
//
// Checkout the current kubernetes configuration status and find the
// artifact spec for the service and branch.
//
// Copy artifacts from the artifacts into the environment and commit the changes.
func (s *Service) ReleaseBranch(ctx context.Context, actor Actor, environment, service, branch string) (string, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.ReleaseBranch")
	defer span.Finish()
	var result string
	err := s.retry(ctx, func(ctx context.Context, attempt int) (bool, error) {
		sourceConfigRepoPath, close, err := git.TempDir(ctx, s.Tracer, "k8s-config-release-branch")
		if err != nil {
			return true, err
		}
		defer close(ctx)
		repo, err := s.Git.Clone(ctx, sourceConfigRepoPath)
		if err != nil {
			return true, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
		}
		// repo/artifacts/{service}/{branch}/{artifactFileName}
		artifactSpecPath := path.Join(artifactPath(sourceConfigRepoPath, service, branch), s.ArtifactFileName)
		artifactSpec, err := artifact.Get(artifactSpecPath)
		if err != nil {
			return true, errors.WithMessage(err, fmt.Sprintf("locate source spec"))
		}
		log.Infof("flow: ReleaseBranch: release branch: id '%s'", artifactSpec.ID)

		// default to environment name for the namespace if none is specified
		namespace := artifactSpec.Namespace
		if namespace == "" {
			namespace = environment
		}

		// release service to env from the artifact path
		// repo/artifacts/{service}/{branch}/{env}
		artifactPath := srcPath(sourceConfigRepoPath, service, branch, environment)
		// repo/{env}/releases/{ns}/{service}
		destinationPath := releasePath(sourceConfigRepoPath, service, environment, namespace)
		log.Infof("flow: ReleaseBranch: copy resources from %s to %s", artifactPath, destinationPath)

		err = s.cleanCopy(ctx, artifactPath, destinationPath)
		if err != nil {
			return true, errors.WithMessagef(err, "copy resources from '%s' to '%s'", artifactPath, destinationPath)
		}

		// copy artifact spec
		// repo/{env}/releases/{ns}/{service}/{artifactFileName}
		artifactDestinationPath := path.Join(releasePath(sourceConfigRepoPath, service, environment, namespace), s.ArtifactFileName)
		log.Infof("flow: ReleaseBranch: copy artifact from %s to %s", artifactSpecPath, artifactDestinationPath)
		err = copy.Copy(artifactSpecPath, artifactDestinationPath)
		if err != nil {
			return true, errors.WithMessage(err, fmt.Sprintf("copy artifact spec from '%s' to '%s'", artifactSpecPath, artifactDestinationPath))
		}

		authorName := artifactSpec.Application.AuthorName
		authorEmail := artifactSpec.Application.AuthorEmail
		artifactID := artifactSpec.ID
		releaseMessage := git.ReleaseCommitMessage(environment, service, artifactID)
		err = s.Git.Commit(ctx, repo, sourceConfigRepoPath, releasePath(".", service, environment, namespace), authorName, authorEmail, actor.Name, actor.Email, releaseMessage)
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
			ArtifactID:    artifactSpec.ID,
			CommitAuthor:  artifactSpec.Application.AuthorName,
			CommitMessage: artifactSpec.Application.Message,
			CommitSHA:     artifactSpec.Application.SHA,
			CommitLink:    artifactSpec.Application.URL,
			Releaser:      actor.Name,
		})
		if err != nil {
			log.Errorf("flow: ReleaseBranch: error notifying release: %v", err)
		}
		result = artifactID
		log.Infof("flow: ReleaseBranch: release committed: %s, Author: %s <%s>, Committer: %s <%s>", releaseMessage, authorName, authorEmail, actor.Name, actor.Email)
		return true, nil
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

// ReleaseArtifactID releases a specific artifact to environment env.
//
// Flow
//
// Locate the commit of the artifact ID and checkout the config repository at
// this point.
//
// Copy resources from the artifact commit into the environment and commit
// the changes
func (s *Service) ReleaseArtifactID(ctx context.Context, actor Actor, environment, service, artifactID string) (string, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.ReleaseArtifactID")
	defer span.Finish()
	var result string
	err := s.retry(ctx, func(ctx context.Context, attempt int) (bool, error) {
		sourceConfigRepoPath, closeSource, err := git.TempDir(ctx, s.Tracer, "k8s-config-release-artifact-source")
		if err != nil {
			return true, err
		}
		defer closeSource(ctx)
		destinationConfigRepoPath, closeDestination, err := git.TempDir(ctx, s.Tracer, "k8s-config-release-artifact-destination")
		if err != nil {
			return true, err
		}
		defer closeDestination(ctx)
		sourceRepo, err := s.Git.Clone(ctx, sourceConfigRepoPath)
		if err != nil {
			return true, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
		}

		hash, err := s.Git.LocateArtifact(ctx, sourceRepo, artifactID)
		if err != nil {
			return true, errors.WithMessagef(err, "locate release '%s'", artifactID)
		}
		err = s.Git.Checkout(ctx, sourceConfigRepoPath, hash)
		if err != nil {
			return true, errors.WithMessagef(err, "checkout release hash '%s'", hash)
		}

		branch, err := git.BranchFromHead(ctx, sourceRepo, s.ArtifactFileName, service)
		if err != nil {
			return true, errors.WithMessagef(err, "locate branch from commit hash '%s'", hash)
		}
		sourceSpec, err := artifact.Get(srcPath(sourceConfigRepoPath, service, branch, s.ArtifactFileName))
		if err != nil {
			return true, errors.WithMessage(err, fmt.Sprintf("locate source spec"))
		}

		log.Infof("flow: ReleaseArtifactID: hash '%s' id '%s'", hash, sourceSpec.ID)

		// default to environment name for the namespace if none is specified
		namespace := sourceSpec.Namespace
		if namespace == "" {
			namespace = environment
		}

		destinationRepo, err := s.Git.Clone(ctx, destinationConfigRepoPath)
		if err != nil {
			return true, errors.WithMessagef(err, "clone destination repo into '%s'", destinationConfigRepoPath)
		}

		// release service to env from original release
		sourcePath := srcPath(sourceConfigRepoPath, service, branch, environment)
		destinationPath := releasePath(destinationConfigRepoPath, service, environment, namespace)
		log.Infof("flow: ReleaseArtifactID: copy resources from %s to %s", sourcePath, destinationPath)

		err = s.cleanCopy(ctx, sourcePath, destinationPath)
		if err != nil {
			return true, errors.WithMessagef(err, "copy resources from '%s' to '%s'", sourcePath, destinationPath)
		}
		// copy artifact spec
		artifactSourcePath := srcPath(sourceConfigRepoPath, service, branch, s.ArtifactFileName)
		artifactDestinationPath := path.Join(releasePath(destinationConfigRepoPath, service, environment, namespace), s.ArtifactFileName)
		log.Infof("flow: ReleaseArtifactID: copy artifact from %s to %s", artifactSourcePath, artifactDestinationPath)
		err = copy.Copy(artifactSourcePath, artifactDestinationPath)
		if err != nil {
			return true, errors.WithMessage(err, fmt.Sprintf("copy artifact spec from '%s' to '%s'", artifactSourcePath, artifactDestinationPath))
		}

		authorName := sourceSpec.Application.AuthorName
		authorEmail := sourceSpec.Application.AuthorEmail
		releaseMessage := git.ReleaseCommitMessage(environment, service, artifactID)
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
			ArtifactID:    sourceSpec.ID,
			CommitAuthor:  sourceSpec.Application.AuthorName,
			CommitMessage: sourceSpec.Application.Message,
			CommitSHA:     sourceSpec.Application.SHA,
			CommitLink:    sourceSpec.Application.URL,
			Releaser:      actor.Name,
		})
		if err != nil {
			log.Errorf("flow: ReleaseBranch: error notifying release: %v", err)
		}
		result = artifactID
		log.Infof("flow: ReleaseArtifactID: release committed: %s, Author: %s <%s>, Committer: %s <%s>", releaseMessage, authorName, authorEmail, actor.Name, actor.Email)
		return true, nil
	})
	if err != nil {
		return "", err
	}
	return result, nil
}
