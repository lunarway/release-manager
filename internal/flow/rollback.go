package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/lunarway/release-manager/internal/copy"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	"gopkg.in/src-d/go-git.v4/plumbing"
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
	sourceConfigRepoPath, closeSource, err := git.TempDirAsync(ctx, s.Tracer, "k8s-config-rollback-source")
	if err != nil {
		return RollbackResult{}, err
	}
	defer closeSource(ctx)

	r, err := s.Git.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
	}

	// locate current release
	currentHash, err := s.Git.LocateServiceRelease(ctx, r, environment, service)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "locate current release at '%s'", sourceConfigRepoPath)
	}
	logger := log.WithContext(ctx)
	logger.Debugf("flow: Rollback: current release hash '%v'", currentHash)
	err = s.Git.Checkout(ctx, sourceConfigRepoPath, currentHash)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "checkout current release hash '%v'", currentHash)
	}
	// default to environment name for the namespace if none is specified
	if namespace == "" {
		namespace = environment
	}
	logger.Infof("flow: Rollback: using namespace '%s'", namespace)

	currentSpec, err := envSpec(sourceConfigRepoPath, s.ArtifactFileName, service, environment, namespace)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "get spec of current release hash '%v'", currentHash)
	}

	// if artifact has no namespace we only allow using the environment as
	// namespace.
	if currentSpec.Namespace == "" && namespace != environment {
		return RollbackResult{}, errors.WithMessagef(ErrNamespaceNotAllowedByArtifact, "namespace '%s'", namespace)
	}

	// a developer can mistakenly specify the wrong namespace when promoting and
	// we will not be able to detect it before this point.
	// It only affects "dev" promotes as we read from the artifacts here where we
	// can find the artifact without taking the namespace into account.
	if currentSpec.Namespace != "" && namespace != currentSpec.Namespace {
		logger.Infof("flow: Rollback: overwriting namespace '%s' to '%s'", namespace, currentSpec.Namespace)
		namespace = currentSpec.Namespace
		result.OverwritingNamespace = currentSpec.Namespace
	}
	// locate new release (the previous released artifact for this service)
	newHash, err := s.Git.LocateServiceReleaseRollbackSkip(ctx, r, environment, service, 1)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "locate previous release at '%s'", sourceConfigRepoPath)
	}
	logger.Debugf("flow: Rollback: new release hash '%v'", newHash)
	err = s.Git.Checkout(ctx, sourceConfigRepoPath, newHash)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "checkout previous release hash '%v'", newHash)
	}

	newSpec, err := envSpec(sourceConfigRepoPath, s.ArtifactFileName, service, environment, namespace)
	if err != nil {
		return RollbackResult{}, errors.WithMessagef(err, "get spec of previous release hash '%v'", newHash)
	}

	err = s.PublishRollback(ctx, RollbackEvent{
		Service:     service,
		NewHash:     newHash.String(),
		Actor:       actor,
		Environment: environment,
		Namespace:   namespace,
	})
	if err != nil {
		return RollbackResult{}, errors.WithMessage(err, "publish event")
	}
	result.Previous = currentSpec.ID
	result.New = newSpec.ID
	return result, nil
}

type RollbackEvent struct {
	Service           string `json:"service,omitempty"`
	Environment       string `json:"environment,omitempty"`
	Namespace         string `json:"namespace,omitempty"`
	CurrentArtifactID string `json:"currentArtifactID,omitempty"`
	NewHash           string `json:"newHash,omitempty"`
	Actor             Actor  `json:"actor,omitempty"`
}

func (RollbackEvent) Type() string {
	return "rollback"
}

func (p RollbackEvent) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *RollbackEvent) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}

func (s *Service) ExecRollback(ctx context.Context, event RollbackEvent) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.ExecRollback")
	defer span.Finish()

	err := s.retry(ctx, func(ctx context.Context, attempt int) (bool, error) {
		logger := log.WithContext(ctx)

		sourceConfigRepoPath, closeSource, err := git.TempDirAsync(ctx, s.Tracer, "k8s-config-rollback-source")
		if err != nil {
			return true, err
		}
		defer closeSource(ctx)

		_, err = s.Git.Clone(ctx, sourceConfigRepoPath)
		if err != nil {
			return true, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
		}

		service := event.Service
		environment := event.Environment
		namespace := event.Namespace
		currentSpecID := event.CurrentArtifactID
		actor := event.Actor
		newHash := plumbing.NewHash(event.NewHash)

		err = s.Git.Checkout(ctx, sourceConfigRepoPath, newHash)
		if err != nil {
			return true, errors.WithMessagef(err, "checkout previous release hash '%v'", newHash)
		}

		destinationConfigRepoPath, closeDestination, err := git.TempDirAsync(ctx, s.Tracer, "k8s-config-rollback-destination")
		if err != nil {
			return true, err
		}
		defer closeDestination(ctx)

		// copy current release artifacts into env
		_, err = s.Git.Clone(ctx, destinationConfigRepoPath)
		if err != nil {
			return true, errors.WithMessagef(err, "clone destination repo into '%s'", destinationConfigRepoPath)
		}

		// release service to env from original release
		sourcePath := releasePath(sourceConfigRepoPath, service, environment, namespace)
		destinationPath := releasePath(destinationConfigRepoPath, service, environment, namespace)
		logger.Infof("flow: ReleaseArtifactID: copy resources from %s to %s", sourcePath, destinationPath)

		err = s.cleanCopy(ctx, sourcePath, destinationPath)
		if err != nil {
			return true, errors.WithMessagef(err, "copy resources from '%s' to '%s'", sourcePath, destinationPath)
		}
		// copy artifact spec
		artifactSourcePath := path.Join(releasePath(sourceConfigRepoPath, service, environment, namespace), s.ArtifactFileName)
		artifactDestinationPath := path.Join(releasePath(destinationConfigRepoPath, service, environment, namespace), s.ArtifactFileName)
		logger.Infof("flow: ReleaseArtifactID: copy artifact from %s to %s", artifactSourcePath, artifactDestinationPath)
		err = copy.CopyFile(ctx, artifactSourcePath, artifactDestinationPath)
		if err != nil {
			return true, errors.WithMessage(err, fmt.Sprintf("copy artifact spec from '%s' to '%s'", artifactSourcePath, artifactDestinationPath))
		}

		newSpec, err := envSpec(sourceConfigRepoPath, s.ArtifactFileName, service, environment, namespace)
		if err != nil {
			return true, errors.WithMessagef(err, "get spec of previous release hash '%v'", newHash)
		}

		authorName := newSpec.Application.AuthorName
		authorEmail := newSpec.Application.AuthorEmail
		releaseMessage := git.RollbackCommitMessage(environment, service, currentSpecID, newSpec.ID)
		err = s.Git.Commit(ctx, destinationConfigRepoPath, releasePath(".", service, environment, namespace), authorName, authorEmail, actor.Name, actor.Email, releaseMessage)
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
			Spec:        newSpec,
			Releaser:    actor.Name,
		})
		logger.Infof("flow: Rollback: rollback committed: %s, Author: %s <%s>, Committer: %s <%s>", releaseMessage, authorName, authorEmail, actor.Name, actor.Email)
		return true, nil
	})
	if err != nil {
		return err
	}
	return nil
}
