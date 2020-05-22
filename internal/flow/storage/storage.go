package storage

import (
	"context"
	"path"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/pkg/errors"
)

type ArtifactLocation struct {
	Branch  string
	Service string
}

type Storage interface {
	ArtifactExists(ctx context.Context, artifactID string) (bool, error)

	ArtifactSpecification(ctx context.Context, service, artifactID string) (artifact.Spec, error)
	ArtifactPaths(ctx context.Context, service, environment, branch, artifactID string) (specPath, resourcesPath string, close CloseFunc, err error)

	LatestArtifactSpecification(context.Context, ArtifactLocation) (artifact.Spec, error)
	LatestArtifactPaths(ctx context.Context, service, environment, branch string) (specPath, resourcesPath string, close CloseFunc, err error)

	ArtifactSpecifications(ctx context.Context, service string, count int) ([]artifact.Spec, error)
}

type CloseFunc func(context.Context)

type Git struct {
	ArtifactFileName string
	Git              *git.Service
	Tracer           tracing.Tracer
}

var _ Storage = &Git{}

func NewGit(artifactFileName string, gitService *git.Service, tracer tracing.Tracer) *Git {
	return &Git{
		ArtifactFileName: artifactFileName,
		Git:              gitService,
		Tracer:           tracer,
	}
}

func (s *Git) ArtifactPaths(ctx context.Context, service, environment, branch, artifactID string) (string, string, CloseFunc, error) {
	logger := log.WithContext(ctx)
	sourceConfigRepoPath, closeSource, err := git.TempDirAsync(ctx, s.Tracer, "k8s-config-promote-source")
	if err != nil {
		return "", "", nil, errors.WithMessage(err, "get temp dir")
	}

	logger.Debugf("Cloning source config repo %s into %s", s.Git.ConfigRepoURL, sourceConfigRepoPath)
	sourceRepo, err := s.Git.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		closeSource(ctx)
		return "", "", nil, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
	}

	hash, err := s.Git.LocateArtifact(ctx, sourceRepo, artifactID)
	if err != nil {
		closeSource(ctx)
		return "", "", nil, errors.WithMessage(err, "locate artifact")
	}

	err = s.Git.Checkout(ctx, sourceConfigRepoPath, hash)
	if err != nil {
		closeSource(ctx)
		return "", "", nil, errors.WithMessagef(err, "checkout artifact hash '%s'", hash)
	}

	resourcesPath := artifactResourcesPath(sourceConfigRepoPath, service, branch, environment)
	specPath := artifactResourcesPath(sourceConfigRepoPath, service, branch, s.ArtifactFileName)
	logger.Infof("storage/ArtifactPaths found resources from '%s' and specification at '%s'", resourcesPath, specPath)
	return specPath, resourcesPath, func(ctx context.Context) {
		closeSource(ctx)
	}, nil
}

func (s *Git) LatestArtifactPaths(ctx context.Context, service, environment, branch string) (string, string, CloseFunc, error) {
	logger := log.WithContext(ctx)
	sourceConfigRepoPath, close, err := git.TempDirAsync(ctx, s.Tracer, "k8s-config-release-branch")
	if err != nil {
		return "", "", nil, err
	}
	_, err = s.Git.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		close(ctx)
		return "", "", nil, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
	}
	resourcesPath := artifactResourcesPath(sourceConfigRepoPath, service, branch, environment)
	specPath := artifactResourcesPath(sourceConfigRepoPath, service, branch, s.ArtifactFileName)
	logger.Infof("storage/LatestArtifactPaths found resources from '%s' and specification at '%s'", resourcesPath, specPath)
	return specPath, resourcesPath, func(ctx context.Context) {
		close(ctx)
	}, nil
}

func (s *Git) LatestArtifactSpecification(ctx context.Context, location ArtifactLocation) (artifact.Spec, error) {
	return artifact.Get(path.Join(artifactSpecPath(s.Git.MasterPath(), location.Service, location.Branch), s.ArtifactFileName))
}

func artifactSpecPath(root, service, branch string) string {
	return path.Join(root, "artifacts", service, branch)
}
func artifactResourcesPath(root, service, branch, env string) string {
	return path.Join(artifactSpecPath(root, service, branch), env)
}

func (s *Git) ArtifactSpecification(ctx context.Context, service, artifactID string) (artifact.Spec, error) {
	sourceConfigRepoPath, close, err := git.TempDirAsync(ctx, s.Tracer, "k8s-config-describe-artifact")
	if err != nil {
		return artifact.Spec{}, err
	}
	defer close(ctx)
	sourceRepo, err := s.Git.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		return artifact.Spec{}, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
	}
	hash, err := s.Git.LocateArtifact(ctx, sourceRepo, artifactID)
	if err != nil {
		return artifact.Spec{}, errors.WithMessage(err, "locate artifact")
	}
	err = s.Git.Checkout(ctx, sourceConfigRepoPath, hash)
	if err != nil {
		return artifact.Spec{}, errors.WithMessagef(err, "checkout hash '%s'", hash)
	}

	branch, err := git.BranchFromHead(ctx, sourceRepo, s.ArtifactFileName, service)
	if err != nil {
		return artifact.Spec{}, errors.WithMessagef(err, "locate branch from commit hash '%s'", hash)
	}

	artifactPath := artifactResourcesPath(sourceConfigRepoPath, service, branch, s.ArtifactFileName)
	spec, err := artifact.Get(artifactPath)
	if err != nil {
		return artifact.Spec{}, errors.WithMessagef(err, "read specification from '%s'", artifactPath)
	}
	log.WithContext(ctx).WithFields("spec", spec).Debugf("Found specifications for service '%s'", service)
	return spec, nil
}

func (s *Git) ArtifactSpecifications(ctx context.Context, service string, count int) ([]artifact.Spec, error) {
	sourceConfigRepoPath, close, err := git.TempDirAsync(ctx, s.Tracer, "k8s-config-describe-artifact")
	if err != nil {
		return nil, err
	}
	defer close(ctx)
	sourceRepo, err := s.Git.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		return nil, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
	}

	hashes, err := s.Git.LocateArtifacts(ctx, sourceRepo, service, count)
	if err != nil {
		return nil, errors.WithMessage(err, "locate artifacts")
	}
	logger := log.WithContext(ctx)
	var artifacts []artifact.Spec
	logger.Debugf("flow/describe: hashes %+v", hashes)
	for _, hash := range hashes {
		err = s.Git.Checkout(ctx, sourceConfigRepoPath, hash)
		if err != nil {
			return nil, errors.WithMessagef(err, "checkout release hash '%s'", hash)
		}
		branch, err := git.BranchFromHead(ctx, sourceRepo, s.ArtifactFileName, service)
		if err != nil {
			logger.Errorf("flow/describe: get branch from head failed at hash '%s': skipping hash: %v", hash, err)
			continue
		}
		artifactPath := path.Join(artifactSpecPath(sourceConfigRepoPath, service, branch), s.ArtifactFileName)
		spec, err := artifact.Get(artifactPath)
		if err != nil {
			return nil, errors.WithMessagef(err, "get artifact at path '%s' at hash '%s'", artifactPath, hash)
		}
		artifacts = append(artifacts, spec)
	}
	return artifacts, nil
}

func (s *Git) ArtifactExists(ctx context.Context, artifactID string) (bool, error) {
	logger := log.WithContext(ctx)
	sourceConfigRepoPath, closeSource, err := git.TempDirAsync(ctx, s.Tracer, "k8s-config-artifact-exists")
	if err != nil {
		return false, err
	}
	defer closeSource(ctx)

	logger.Debugf("Cloning source config repo %s into %s", s.Git.ConfigRepoURL, sourceConfigRepoPath)
	sourceRepo, err := s.Git.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		return false, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
	}
	_, err = s.Git.LocateArtifact(ctx, sourceRepo, artifactID)
	if err != nil {
		return false, errors.WithMessage(err, "locate artifact")
	}
	return true, nil
}
