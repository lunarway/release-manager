package flow

import (
	"context"
	"fmt"
	"path"

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
		sourceConfigRepoPath, closeSource, err := git.TempDirAsync(ctx, s.Tracer, "k8s-config-promote-source")
		if err != nil {
			return true, err
		}
		defer closeSource(ctx)

		logger := log.WithContext(ctx)
		// find current released artifact.json for service in env - 1 (dev for staging, staging for prod)
		logger.Debugf("Cloning source config repo %s into %s", s.Git.ConfigRepoURL, sourceConfigRepoPath)
		sourceRepo, err := s.Git.Clone(ctx, sourceConfigRepoPath)
		if err != nil {
			return true, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
		}

		// default to environment name for the namespace if none is specified
		if namespace == "" {
			namespace = environment
		}
		logger.Infof("flow: Promote: using namespace '%s'", namespace)

		sourceSpec, err := sourceSpec(ctx, sourceConfigRepoPath, s.ArtifactFileName, service, environment, namespace)
		if err != nil {
			return true, errors.WithMessage(err, fmt.Sprintf("locate source spec"))
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
			hash, err = s.Git.LocateArtifact(ctx, sourceRepo, result.ReleaseID)
		} else {
			hash, err = s.Git.LocateRelease(ctx, sourceRepo, result.ReleaseID)
		}
		if err != nil {
			return true, errors.WithMessagef(err, "locate release '%s'", result.ReleaseID)
		}

		// TODO: we don't know if there is any changes to commit at this point, but
		// it might be nice to be able to answer that directly. we could check the
		// current artifact ID in the environment right away?

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

func (p PromoteEvent) Body() interface{} {
	return p
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
	span, ctx := s.Tracer.FromCtx(ctx, "flow.Promote")
	defer span.Finish()
	var result PromoteResult
	err := s.retry(ctx, func(ctx context.Context, attempt int) (bool, error) {
		logger := log.WithContext(ctx)

		sourceConfigRepoPath, closeSource, err := git.TempDirAsync(ctx, s.Tracer, "k8s-config-promote-source")
		if err != nil {
			return true, err
		}
		defer closeSource(ctx)

		// find current released artifact.json for service in env - 1 (dev for staging, staging for prod)
		logger.Debugf("Cloning source config repo %s into %s", s.Git.ConfigRepoURL, sourceConfigRepoPath)
		_, err = s.Git.Clone(ctx, sourceConfigRepoPath)
		if err != nil {
			return true, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
		}

		hash := plumbing.NewHash(p.Hash)
		service := p.Service
		environment := p.Environment
		namespace := p.Namespace
		actor := p.Actor
		logger.Debugf("internal/flow: Promote: release hash '%v'", hash)
		err = s.Git.Checkout(ctx, sourceConfigRepoPath, hash)
		if err != nil {
			return true, errors.WithMessagef(err, "checkout release hash '%s'", hash)
		}

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
		sourcePath := srcPath(sourceConfigRepoPath, service, "master", environment)
		destinationPath := releasePath(destinationConfigRepoPath, service, environment, namespace)
		logger.Debugf("Copy resources from: %s to %s", sourcePath, destinationPath)

		err = s.cleanCopy(ctx, sourcePath, destinationPath)
		if err != nil {
			return true, errors.WithMessagef(err, "copy resources from '%s' to '%s'", sourcePath, destinationPath)
		}

		// copy artifact spec
		artifactSourcePath := srcPath(sourceConfigRepoPath, service, "master", s.ArtifactFileName)
		artifactDestinationPath := path.Join(releasePath(destinationConfigRepoPath, service, environment, namespace), s.ArtifactFileName)
		logger.Debugf("Copy artifact from: %s to %s", artifactSourcePath, artifactDestinationPath)
		err = copy.CopyFile(ctx, artifactSourcePath, artifactDestinationPath)
		if err != nil {
			return true, errors.WithMessage(err, fmt.Sprintf("copy artifact spec from '%s' to '%s'", artifactSourcePath, artifactDestinationPath))
		}

		sourceSpec, err := sourceSpec(ctx, sourceConfigRepoPath, s.ArtifactFileName, service, environment, namespace)
		if err != nil {
			return true, errors.WithMessage(err, fmt.Sprintf("locate source spec"))
		}
		authorName := sourceSpec.Application.AuthorName
		authorEmail := sourceSpec.Application.AuthorEmail
		releaseMessage := git.ReleaseCommitMessage(environment, service, result.ReleaseID)
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
