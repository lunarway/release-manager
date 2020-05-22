package git

import (
	"context"
	"path"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

func (s *Service) ArtifactPaths(ctx context.Context, service, environment, branch, artifactID string) (string, string, func(context.Context), error) {
	logger := log.WithContext(ctx)
	sourceConfigRepoPath, closeSource, err := TempDirAsync(ctx, s.Tracer, "k8s-config-artifact-paths-source")
	if err != nil {
		return "", "", nil, errors.WithMessage(err, "get temp dir")
	}

	logger.Debugf("Cloning source config repo %s into %s", s.ConfigRepoURL, sourceConfigRepoPath)
	sourceRepo, err := s.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		closeSource(ctx)
		return "", "", nil, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
	}

	hash, err := s.LocateArtifact(ctx, sourceRepo, artifactID)
	if err != nil {
		closeSource(ctx)
		return "", "", nil, errors.WithMessage(err, "locate artifact")
	}

	err = s.Checkout(ctx, sourceConfigRepoPath, hash)
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

func (s *Service) LatestArtifactPaths(ctx context.Context, service, environment, branch string) (string, string, func(context.Context), error) {
	logger := log.WithContext(ctx)
	sourceConfigRepoPath, close, err := TempDirAsync(ctx, s.Tracer, "k8s-config-release-branch")
	if err != nil {
		return "", "", nil, err
	}
	_, err = s.Clone(ctx, sourceConfigRepoPath)
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

func (s *Service) LatestArtifactSpecification(ctx context.Context, service, branch string) (artifact.Spec, error) {
	return artifact.Get(path.Join(artifactSpecPath(s.MasterPath(), service, branch), s.ArtifactFileName))
}

func artifactSpecPath(root, service, branch string) string {
	return path.Join(root, "artifacts", service, branch)
}
func artifactResourcesPath(root, service, branch, env string) string {
	return path.Join(artifactSpecPath(root, service, branch), env)
}

func (s *Service) ArtifactSpecification(ctx context.Context, service, artifactID string) (artifact.Spec, error) {
	sourceConfigRepoPath, close, err := TempDirAsync(ctx, s.Tracer, "k8s-config-describe-artifact")
	if err != nil {
		return artifact.Spec{}, err
	}
	defer close(ctx)
	sourceRepo, err := s.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		return artifact.Spec{}, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
	}
	hash, err := s.LocateArtifact(ctx, sourceRepo, artifactID)
	if err != nil {
		return artifact.Spec{}, errors.WithMessage(err, "locate artifact")
	}
	err = s.Checkout(ctx, sourceConfigRepoPath, hash)
	if err != nil {
		return artifact.Spec{}, errors.WithMessagef(err, "checkout hash '%s'", hash)
	}

	branch, err := BranchFromHead(ctx, sourceRepo, s.ArtifactFileName, service)
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

func (s *Service) ArtifactSpecifications(ctx context.Context, service string, count int) ([]artifact.Spec, error) {
	sourceConfigRepoPath, close, err := TempDirAsync(ctx, s.Tracer, "k8s-config-describe-artifact")
	if err != nil {
		return nil, err
	}
	defer close(ctx)
	sourceRepo, err := s.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		return nil, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
	}

	hashes, err := s.LocateArtifacts(ctx, sourceRepo, service, count)
	if err != nil {
		return nil, errors.WithMessage(err, "locate artifacts")
	}
	logger := log.WithContext(ctx)
	var artifacts []artifact.Spec
	logger.Debugf("flow/describe: hashes %+v", hashes)
	for _, hash := range hashes {
		err = s.Checkout(ctx, sourceConfigRepoPath, hash)
		if err != nil {
			return nil, errors.WithMessagef(err, "checkout release hash '%s'", hash)
		}
		branch, err := BranchFromHead(ctx, sourceRepo, s.ArtifactFileName, service)
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

func (s *Service) ArtifactExists(ctx context.Context, artifactID string) (bool, error) {
	logger := log.WithContext(ctx)
	sourceConfigRepoPath, closeSource, err := TempDirAsync(ctx, s.Tracer, "k8s-config-artifact-exists")
	if err != nil {
		return false, err
	}
	defer closeSource(ctx)

	logger.Debugf("Cloning source config repo %s into %s", s.ConfigRepoURL, sourceConfigRepoPath)
	sourceRepo, err := s.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		return false, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
	}
	_, err = s.LocateArtifact(ctx, sourceRepo, artifactID)
	if err != nil {
		return false, errors.WithMessage(err, "locate artifact")
	}
	return true, nil
}
