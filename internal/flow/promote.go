package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/copy"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// Promote promotes a specific service to environment env. The flow is async in
// that this method validates the inputs and publishes an event that is handled
// later on by ExecPromote.
func (s *Service) Promote(ctx context.Context, actor Actor, environment, namespace, service string) (PromoteResult, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.Promote")
	defer span.Finish()
	var result PromoteResult
	err := s.retry(ctx, func(ctx context.Context, attempt int) (bool, error) {
		logger := log.WithContext(ctx)

		// default to environment name for the namespace if none is specified
		if namespace == "" {
			namespace = environment
		}
		logger.Infof("flow: Promote: using namespace '%s'", namespace)

		// locate the previous environment
		sourceSpec, err := s.sourceSpec(ctx, service, environment, namespace)
		if err != nil {
			return true, errors.WithMessage(err, fmt.Sprintf("locate source spec"))
		}

		ok, err := s.CanRelease(ctx, service, sourceSpec.Application.Branch, environment)
		if err != nil {
			return true, errors.WithMessage(err, "validate release policies")
		}
		if !ok {
			return true, ErrReleaseProhibited
		}

		// if artifact has no namespace we only allow using the environment as
		// namespace.
		if sourceSpec.Namespace == "" && namespace != environment {
			return true, errors.WithMessagef(ErrNamespaceNotAllowedByArtifact, "namespace '%s'", namespace)
		}

		// a developer can mistakenly specify the wrong namespace when promoting and
		// we will not be able to detect it before this point.
		// It only affects "dev" promotes as we read from the artifacts here where we
		// can find the artifact without taking the namespace into account.
		if sourceSpec.Namespace != "" && namespace != sourceSpec.Namespace {
			logger.Infof("flow: Promote: overwriting namespace '%s' to '%s'", namespace, sourceSpec.Namespace)
			namespace = sourceSpec.Namespace
			result.OverwritingNamespace = sourceSpec.Namespace
		}

		// find release identifier in artifact.json
		result.ReleaseID = sourceSpec.ID
		// ckechout commit of release
		var hash plumbing.Hash
		// when promoting to dev we use should look for the artifact instead of
		// release as the artifact have never been released.
		if environment == "dev" {
			hash, err = s.Storage.GetHashForArtifact(ctx, result.ReleaseID)
		} else {
			hash, err = s.getHashForRelease(ctx, result.ReleaseID)
		}
		if err != nil {
			return true, errors.WithMessagef(err, "locate release '%s'", result.ReleaseID)
		}

		// check that the artifact to be released is not already released in the
		// environment. If there is no artifact released to the target environment an
		// artifact.ErrFileNotFound error is returned. This is OK as the currentSpec
		// will then be the default value and this its ID will be the empty string.
		currentSpec, err := s.releaseSpecification(ctx, releaseLocation{
			Environment: environment,
			Namespace:   namespace,
			Service:     service,
		})
		// currentSpec, err := envSpec(sourceConfigRepoPath, s.ArtifactFileName, service, environment, namespace)
		if err != nil && errors.Cause(err) != artifact.ErrFileNotFound {
			return true, errors.WithMessage(err, "get current released spec")
		}
		if currentSpec.ID == sourceSpec.ID {
			return true, ErrNothingToRelease
		}

		err = s.PublishPromote(ctx, PromoteEvent{
			Hash:        hash.String(),
			Service:     service,
			Environment: environment,
			Namespace:   namespace,
			Actor:       actor,
		})
		if err != nil {
			return true, errors.WithMessage(err, "publish message")
		}
		return true, nil
	})
	if err != nil {
		return PromoteResult{}, err
	}
	return result, nil
}

type PromoteResult struct {
	ReleaseID            string
	OverwritingNamespace string
}

type PromoteEvent struct {
	Hash        string `json:"hash,omitempty"`
	Service     string `json:"service,omitempty"`
	Environment string `json:"environment,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Actor       Actor  `json:"actor,omitempty"`
}

func (PromoteEvent) Type() string {
	return "promote"
}

func (p PromoteEvent) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

func (p *PromoteEvent) Unmarshal(data []byte) error {
	return json.Unmarshal(data, p)
}

// ExecPromote promotes a specific service to environment env.
//
// By convention, promotion means:
//
//    Move released version of the previous environment into this environment
//
// Promotion follows this convention
//
//   master -> dev -> staging -> prod
//
// Flow
//
// Checkout the current kubernetes configuration status and find the
// artifact.json spec for the service and previous environment.
// Use the artifact ID as a key for locating the artifacts.
//
// Find the commit with the artifact ID and checkout the config repository at
// this point.
//
// Copy artifacts from the current release into the new environment and commit
// the changes
func (s *Service) ExecPromote(ctx context.Context, p PromoteEvent) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.ExecPromote")
	defer span.Finish()
	err := s.retry(ctx, func(ctx context.Context, attempt int) (bool, error) {
		logger := log.WithContext(ctx)

		artifactSourcePath, sourcePath, closeSource, err := s.getArtifactPathFromHash(ctx, p.Hash, p.Service, p.Environment)
		if err != nil {
			return true, err
		}
		defer closeSource(ctx)

		hash := plumbing.NewHash(p.Hash)
		service := p.Service
		environment := p.Environment
		namespace := p.Namespace
		actor := p.Actor
		logger.Debugf("internal/flow: Promote: release hash '%v'", hash)

		destinationConfigRepoPath, closeDest, err := git.TempDirAsync(ctx, s.Tracer, "k8s-config-promote-dest")
		if err != nil {
			return true, err
		}
		defer closeDest(ctx)

		_, err = s.Git.Clone(ctx, destinationConfigRepoPath)
		if err != nil {
			return true, errors.WithMessagef(err, "clone destination repo into '%s'", destinationConfigRepoPath)
		}

		// release service to env from original release
		destinationPath := releasePath(destinationConfigRepoPath, service, environment, namespace)
		logger.Debugf("Copy resources from: %s to %s", sourcePath, destinationPath)

		err = s.cleanCopy(ctx, sourcePath, destinationPath)
		if err != nil {
			return true, errors.WithMessagef(err, "copy resources from '%s' to '%s'", sourcePath, destinationPath)
		}

		// copy artifact spec
		artifactDestinationPath := path.Join(releasePath(destinationConfigRepoPath, service, environment, namespace), s.ArtifactFileName)
		logger.Debugf("Copy artifact from: %s to %s", artifactSourcePath, artifactDestinationPath)
		err = copy.CopyFile(ctx, artifactSourcePath, artifactDestinationPath)
		if err != nil {
			return true, errors.WithMessage(err, fmt.Sprintf("copy artifact spec from '%s' to '%s'", artifactSourcePath, artifactDestinationPath))
		}

		sourceSpec, err := artifact.Get(artifactSourcePath)
		if err != nil {
			return true, errors.WithMessage(err, fmt.Sprintf("locate source spec"))
		}
		authorName := sourceSpec.Application.AuthorName
		authorEmail := sourceSpec.Application.AuthorEmail
		releaseMessage := git.ReleaseCommitMessage(environment, service, sourceSpec.ID, authorEmail)
		logger.Debugf("Committing release: %s, Author: %s <%s>, Committer: %s <%s>", releaseMessage, authorName, authorEmail, actor.Name, actor.Email)
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
			Spec:        sourceSpec,
			Releaser:    actor.Name,
		})
		return true, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) getHashForRelease(ctx context.Context, artifactID string) (plumbing.Hash, error) {
	logger := log.WithContext(ctx)
	sourceConfigRepoPath, closeSource, err := git.TempDirAsync(ctx, s.Tracer, "k8s-config-promote-source")
	if err != nil {
		return plumbing.ZeroHash, err
	}
	defer closeSource(ctx)
	logger.Debugf("Cloning source config repo %s into %s", s.Git.ConfigRepoURL, sourceConfigRepoPath)
	sourceRepo, err := s.Git.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		return plumbing.ZeroHash, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
	}
	hash, err := s.Git.LocateRelease(ctx, sourceRepo, artifactID)
	if err != nil {
		return plumbing.ZeroHash, errors.WithMessage(err, "locate artifact")
	}
	return hash, nil
}

func (s *Service) getArtifactPathFromHash(ctx context.Context, hashStr, service, environment string) (string, string, func(context.Context), error) {
	logger := log.WithContext(ctx)
	sourceConfigRepoPath, closeSource, err := git.TempDirAsync(ctx, s.Tracer, "k8s-config-promote-source")
	if err != nil {
		return "", "", nil, err
	}

	// find current released artifact.json for service in env - 1 (dev for staging, staging for prod)
	logger.Debugf("Cloning source config repo %s into %s", s.Git.ConfigRepoURL, sourceConfigRepoPath)
	_, err = s.Git.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		closeSource(ctx)
		return "", "", nil, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
	}

	hash := plumbing.NewHash(hashStr)
	logger.Debugf("internal/flow: getArtifactPathFromHash: release hash '%v'", hash)
	err = s.Git.Checkout(ctx, sourceConfigRepoPath, hash)
	if err != nil {
		closeSource(ctx)
		return "", "", nil, errors.WithMessagef(err, "checkout release hash '%s'", hash)
	}

	resourcesPath := srcPath(sourceConfigRepoPath, service, "master", environment)
	specPath := srcPath(sourceConfigRepoPath, service, "master", s.ArtifactFileName)
	logger.Infof("internal/flow: getArtifactPathFromHash: found resources from '%s' and specification at '%s'", resourcesPath, specPath)
	return specPath, resourcesPath, func(ctx context.Context) {
		closeSource(ctx)
	}, nil
}
