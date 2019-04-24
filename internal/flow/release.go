package flow

import (
	"context"
	"fmt"
	"os"
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
	sourceConfigRepoPath, close, err := tempDir("k8s-config-release-branch")
	if err != nil {
		return "", err
	}
	defer close()
	repo, err := git.CloneDepth(ctx, s.ConfigRepoURL, sourceConfigRepoPath, s.SSHPrivateKeyPath, 1)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("clone '%s' into '%s'", s.ConfigRepoURL, sourceConfigRepoPath))
	}
	// repo/artifacts/{service}/{branch}/{artifactFileName}
	artifactSpecPath := path.Join(artifactPath(sourceConfigRepoPath, service, branch), s.ArtifactFileName)
	artifactSpec, err := artifact.Get(artifactSpecPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("locate source spec"))
	}
	log.Infof("flow: ReleaseBranch: release branch: id '%s'", artifactSpec.ID)

	// release service to env from the artifact path
	// repo/artifacts/{service}/{branch}/{env}
	artifactPath := srcPath(sourceConfigRepoPath, service, branch, environment)
	// repo/{env}/releases/{ns}/{service}
	destinationPath := releasePath(sourceConfigRepoPath, service, environment)
	log.Infof("flow: ReleaseBranch: copy resources from %s to %s", artifactPath, destinationPath)

	// empty existing resources in destination
	err = os.RemoveAll(destinationPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("remove destination path '%s'", destinationPath))
	}
	err = os.MkdirAll(destinationPath, os.ModePerm)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("create destination dir '%s'", destinationPath))
	}
	// copy artifact files into destination
	err = copy.Copy(artifactPath, destinationPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("copy resources from '%s' to '%s'", artifactPath, destinationPath))
	}
	// copy artifact spec
	// repo/{env}/releases/{ns}/{service}/{artifactFileName}
	artifactDestinationPath := path.Join(releasePath(sourceConfigRepoPath, service, environment), s.ArtifactFileName)
	log.Infof("flow: ReleaseBranch: copy artifact from %s to %s", artifactSpecPath, artifactDestinationPath)
	err = copy.Copy(artifactSpecPath, artifactDestinationPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("copy artifact spec from '%s' to '%s'", artifactSpecPath, artifactDestinationPath))
	}

	authorName := artifactSpec.Application.AuthorName
	authorEmail := artifactSpec.Application.AuthorEmail
	artifactID := artifactSpec.ID
	releaseMessage := git.ReleaseCommitMessage(environment, service, artifactID)
	err = git.Commit(ctx, repo, releasePath(".", service, environment), authorName, authorEmail, actor.Name, actor.Email, releaseMessage, s.SSHPrivateKeyPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("commit changes from path '%s'", destinationPath))
	}
	err = s.notifyRelease(NotifyReleaseOptions{
		Service:       service,
		Environment:   environment,
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
	log.Infof("flow: ReleaseBranch: release committed: %s, Author: %s <%s>, Committer: %s <%s>", releaseMessage, authorName, authorEmail, actor.Name, actor.Email)
	return artifactID, nil
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
	sourceConfigRepoPath, closeSource, err := tempDir("k8s-config-release-artifact-source")
	if err != nil {
		return "", err
	}
	defer closeSource()
	destinationConfigRepoPath, closeDestination, err := tempDir("k8s-config-release-artifact-destination")
	if err != nil {
		return "", err
	}
	defer closeDestination()
	sourceRepo, err := git.Clone(ctx, s.ConfigRepoURL, sourceConfigRepoPath, s.SSHPrivateKeyPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("clone '%s' into '%s'", s.ConfigRepoURL, sourceConfigRepoPath))
	}

	hash, err := git.LocateArtifact(sourceRepo, artifactID)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("locate release '%s' from '%s'", artifactID, s.ConfigRepoURL))
	}
	err = git.Checkout(sourceRepo, hash)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("checkout release hash '%s' from '%s'", hash, s.ConfigRepoURL))
	}

	sourceSpec, err := sourceSpec(sourceConfigRepoPath, s.ArtifactFileName, service, environment)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("locate source spec"))
	}

	log.Infof("flow: ReleaseArtifactID: hash '%s' id '%s'", hash, sourceSpec.ID)

	destinationRepo, err := git.Clone(ctx, s.ConfigRepoURL, destinationConfigRepoPath, s.SSHPrivateKeyPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("clone destination repo '%s' into '%s'", s.ConfigRepoURL, destinationConfigRepoPath))
	}

	// release service to env from original release
	sourcePath := srcPath(sourceConfigRepoPath, service, "master", environment)
	destinationPath := releasePath(destinationConfigRepoPath, service, environment)
	log.Infof("flow: ReleaseArtifactID: copy resources from %s to %s", sourcePath, destinationPath)

	// empty existing resources in destination
	err = os.RemoveAll(destinationPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("remove destination path '%s'", destinationPath))
	}
	err = os.MkdirAll(destinationPath, os.ModePerm)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("create destination dir '%s'", destinationPath))
	}
	// copy previous env. files into destination
	err = copy.Copy(sourcePath, destinationPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("copy resources from '%s' to '%s'", sourcePath, destinationPath))
	}
	// copy artifact spec
	artifactSourcePath := srcPath(sourceConfigRepoPath, service, "master", s.ArtifactFileName)
	artifactDestinationPath := path.Join(releasePath(destinationConfigRepoPath, service, environment), s.ArtifactFileName)
	log.Infof("flow: ReleaseArtifactID: copy artifact from %s to %s", artifactSourcePath, artifactDestinationPath)
	err = copy.Copy(artifactSourcePath, artifactDestinationPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("copy artifact spec from '%s' to '%s'", artifactSourcePath, artifactDestinationPath))
	}

	authorName := sourceSpec.Application.AuthorName
	authorEmail := sourceSpec.Application.AuthorEmail
	releaseMessage := git.ReleaseCommitMessage(environment, service, artifactID)
	err = git.Commit(ctx, destinationRepo, releasePath(".", service, environment), authorName, authorEmail, actor.Name, actor.Email, releaseMessage, s.SSHPrivateKeyPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("commit changes from path '%s'", destinationPath))
	}
	err = s.notifyRelease(NotifyReleaseOptions{
		Service:       service,
		Environment:   environment,
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
	log.Infof("flow: ReleaseArtifactID: release committed: %s, Author: %s <%s>, Committer: %s <%s>", releaseMessage, authorName, authorEmail, actor.Name, actor.Email)

	return artifactID, nil
}
