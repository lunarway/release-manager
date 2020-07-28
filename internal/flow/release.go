package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/commitinfo"
	"github.com/lunarway/release-manager/internal/copy"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/intent"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

// releaseConfigurationExists verifies that the provided path exists. Useful to
// ensure that a given configuration is available for an environment.
func releaseConfigurationExists(resourcePath string) error {
	f, err := os.Open(resourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrUnknownEnvironment
		}
		return err
	}
	defer f.Close()
	return nil
}

type ReleaseArtifactIDEvent struct {
	Service     string        `json:"service,omitempty"`
	Environment string        `json:"environment,omitempty"`
	Namespace   string        `json:"namespace,omitempty"`
	ArtifactID  string        `json:"artifactID,omitempty"`
	Branch      string        `json:"branch,omitempty"`
	Actor       Actor         `json:"actor,omitempty"`
	Intent      intent.Intent `json:"intent,omitempty"`
}

func (ReleaseArtifactIDEvent) Type() string {
	return "release.artifactId"
}

func (p ReleaseArtifactIDEvent) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *ReleaseArtifactIDEvent) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
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
func (s *Service) ReleaseArtifactID(ctx context.Context, actor Actor, environment, service, artifactID string, intent intent.Intent) (string, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.ReleaseArtifactID")
	defer span.Finish()

	sourceSpec, err := s.Storage.ArtifactSpecification(ctx, service, artifactID)
	if err != nil {
		return "", errors.WithMessage(err, "get artifact specification")
	}
	branch := sourceSpec.Application.Branch

	ok, err := s.CanRelease(ctx, service, branch, environment)
	if err != nil {
		return "", errors.WithMessage(err, "validate release policies")
	}
	if !ok {
		return "", ErrReleaseProhibited
	}

	logger := log.WithContext(ctx)
	logger.Infof("flow: ReleaseArtifactID: id '%s'", sourceSpec.ID)

	_, resourcePath, close, err := s.Storage.LatestArtifactPaths(ctx, service, environment, branch)
	if err != nil {
		return "", errors.WithMessage(err, "get artifact paths")
	}
	defer close(ctx)

	// Verify environment existences
	err = releaseConfigurationExists(resourcePath)
	if err != nil {
		return "", errors.WithMessagef(err, "verify configuration for environment in '%s'", resourcePath)
	}

	// default to environment name for the namespace if none is specified
	namespace := sourceSpec.Namespace
	if namespace == "" {
		namespace = environment
	}

	// check that the artifact to be released is not already released in the
	// environment. If there is no artifact released to the target environment an
	// artifact.ErrFileNotFound error is returned. This is OK as the currentSpec
	// will then be the default value and this its ID will be the empty string.
	destinationConfigRepoPath, closeDestinationSource, err := git.TempDirAsync(ctx, s.Tracer, "k8s-config-release-artifact-destination")
	if err != nil {
		return "", err
	}
	defer closeDestinationSource(ctx)
	_, err = s.Git.Clone(ctx, destinationConfigRepoPath)
	if err != nil {
		return "", errors.WithMessagef(err, "clone into '%s'", destinationConfigRepoPath)
	}
	currentSpec, err := envSpec(destinationConfigRepoPath, s.ArtifactFileName, service, environment, namespace)
	if err != nil && errors.Cause(err) != artifact.ErrFileNotFound {
		return "", errors.WithMessage(err, "get current released spec")
	}
	logger.WithFields("currentSpec", currentSpec).Debugf("Found artifact '%s' in environment '%s'", currentSpec.ID, environment)
	if currentSpec.ID == sourceSpec.ID {
		return "", ErrNothingToRelease
	}

	err = s.PublishReleaseArtifactID(ctx, ReleaseArtifactIDEvent{
		ArtifactID:  artifactID,
		Actor:       actor,
		Branch:      branch,
		Environment: environment,
		Namespace:   namespace,
		Service:     service,
		Intent:      intent,
	})
	if err != nil {
		return "", errors.WithMessage(err, "publish event")
	}
	return artifactID, nil
}

func (s *Service) ExecReleaseArtifactID(ctx context.Context, event ReleaseArtifactIDEvent) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.ExecReleaseArtifactID")
	defer span.Finish()
	err := s.retry(ctx, func(ctx context.Context, attempt int) (bool, error) {
		service := event.Service
		branch := event.Branch
		environment := event.Environment
		namespace := event.Namespace
		actor := event.Actor
		artifactID := event.ArtifactID

		logger := log.WithContext(ctx)

		artifactSourcePath, sourcePath, closeSource, err := s.Storage.ArtifactPaths(ctx, service, environment, branch, artifactID)
		if err != nil {
			return true, errors.WithMessage(err, "get artifact paths")
		}
		defer closeSource(ctx)

		destinationConfigRepoPath, closeDestination, err := git.TempDirAsync(ctx, s.Tracer, "k8s-config-release-artifact-destination")
		if err != nil {
			return true, err
		}
		defer closeDestination(ctx)

		_, err = s.Git.Clone(ctx, destinationConfigRepoPath)
		if err != nil {
			return true, errors.WithMessagef(err, "clone destination repo into '%s'", destinationConfigRepoPath)
		}

		// release service to env from original release
		destinationPath, err := releasePath(destinationConfigRepoPath, service, environment, namespace)
		if err != nil {
			return true, errors.WithMessage(err, "get release path")
		}
		logger.Infof("flow: ReleaseArtifactID: copy resources from %s to %s", sourcePath, destinationPath)

		err = s.cleanCopy(ctx, sourcePath, destinationPath)
		if err != nil {
			return true, errors.WithMessagef(err, "copy resources from '%s' to '%s'", sourcePath, destinationPath)
		}
		// copy artifact spec
		artifactDestinationPath := path.Join(destinationPath, s.ArtifactFileName)
		logger.Infof("flow: ReleaseArtifactID: copy artifact from %s to %s", artifactSourcePath, artifactDestinationPath)
		err = copy.CopyFile(ctx, artifactSourcePath, artifactDestinationPath)
		if err != nil {
			return true, errors.WithMessage(err, fmt.Sprintf("copy artifact spec from '%s' to '%s'", artifactSourcePath, artifactDestinationPath))
		}
		sourceSpec, err := artifact.Get(artifactSourcePath)
		if err != nil {
			return true, errors.WithMessage(err, fmt.Sprintf("locate source spec"))
		}
		artifactAuthor := commitinfo.NewPersonInfo(sourceSpec.Application.AuthorName, sourceSpec.Application.AuthorEmail)
		releaseAuthor := commitinfo.NewPersonInfo(actor.Name, actor.Email)
		releaseMessage := commitinfo.ReleaseCommitMessage(environment, service, artifactID, event.Intent, artifactAuthor, releaseAuthor)
		commitPath, err := releasePath(".", service, environment, namespace)
		if err != nil {
			return true, errors.WithMessage(err, "get commit path")
		}
		err = s.Git.Commit(ctx, destinationConfigRepoPath, commitPath, releaseMessage)
		if err != nil {
			if errors.Cause(err) == git.ErrNothingToCommit {
				logger.Infof("Environment is up to date: dropping event: %v", err)
				// TODO: notify actor that there was nothing to commit
				return true, nil
			}
			// we can see races here where other changes are committed to the master repo
			// after we cloned. Because of this we retry on any error.
			return false, errors.WithMessage(err, fmt.Sprintf("commit changes from path '%s'", destinationPath))
		}
		s.notifyRelease(ctx, NotifyReleaseOptions{
			Service:     service,
			Environment: environment,
			Namespace:   namespace,
			Spec:        sourceSpec,
			Releaser:    actor.Name,
		})
		logger.Infof("flow: ReleaseArtifactID: release committed: %s, ArtifactAuthor: %s, ReleaseAuthor: %s", releaseMessage, artifactAuthor, releaseAuthor)
		return true, nil
	})
	if err != nil {
		return err
	}
	return nil
}
