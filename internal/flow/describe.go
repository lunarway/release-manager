package flow

import (
	"context"
	"path"
	"time"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

type DescribeReleaseResponse struct {
	DefaultNamespaces bool
	Artifact          artifact.Spec
	ReleasedAt        time.Time
	ReleasedByEmail   string
	ReleasedByName    string
}

// DescribeRelease returns information about a specific release in an environment.
func (s *Service) DescribeRelease(ctx context.Context, namespace, environment, service string) (DescribeReleaseResponse, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.DescribeRelease")
	defer span.Finish()
	sourceConfigRepoPath, close, err := git.TempDir(ctx, s.Tracer, "k8s-config-describe-release")
	if err != nil {
		return DescribeReleaseResponse{}, err
	}
	defer close(ctx)

	log.Debugf("Cloning source config repo %s into %s", s.Git.ConfigRepoURL, sourceConfigRepoPath)
	sourceRepo, err := s.Git.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		return DescribeReleaseResponse{}, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
	}

	defaultNamespaces := namespace == ""
	if defaultNamespaces {
		namespace = environment
	}

	spec, err := envSpec(sourceConfigRepoPath, s.ArtifactFileName, service, environment, namespace)
	if err != nil {
		return DescribeReleaseResponse{}, errors.WithMessagef(err, "namespace '%s'", namespace)
	}

	hash, err := s.Git.LocateServiceReleaseRollbackSkip(ctx, sourceRepo, environment, service, 0)
	if err != nil {
		return DescribeReleaseResponse{}, errors.WithMessagef(err, "namespace '%s': locate latest release", namespace)
	}
	c, err := sourceRepo.CommitObject(hash)
	if err != nil {
		return DescribeReleaseResponse{}, errors.WithMessagef(err, "namespace '%s': get commit at hash '%s'", namespace, hash)
	}
	return DescribeReleaseResponse{
		DefaultNamespaces: defaultNamespaces,
		Artifact:          spec,
		ReleasedAt:        c.Committer.When,
		ReleasedByEmail:   c.Committer.Email,
		ReleasedByName:    c.Committer.Name,
	}, nil
}

// DescribeArtifact returns n artifacts for a service.
func (s *Service) DescribeArtifact(ctx context.Context, service string, n int) ([]artifact.Spec, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.DescribeArtifact")
	defer span.Finish()
	sourceConfigRepoPath, close, err := git.TempDir(ctx, s.Tracer, "k8s-config-describe-artifact")
	if err != nil {
		return nil, err
	}
	defer close(ctx)

	log.Debugf("Cloning source config repo %s into %s", s.Git.ConfigRepoURL, sourceConfigRepoPath)
	sourceRepo, err := s.Git.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		return nil, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
	}

	hashes, err := s.Git.LocateArtifacts(ctx, sourceRepo, service, n)
	if err != nil {
		return nil, errors.WithMessage(err, "locate artifacts")
	}
	var artifacts []artifact.Spec
	log.Debugf("flow/describe: hashes %+v", hashes)
	for _, hash := range hashes {
		err = s.Git.Checkout(ctx, sourceRepo, hash)
		if err != nil {
			return nil, errors.WithMessagef(err, "checkout release hash '%s'", hash)
		}
		branch, err := git.BranchFromHead(ctx, sourceRepo, s.ArtifactFileName, service)
		if err != nil {
			log.Errorf("flow/describe: get branch from head failed at hash '%s': skipping hash: %v", hash, err)
			continue
		}
		artifactPath := path.Join(artifactPath(sourceConfigRepoPath, service, branch), s.ArtifactFileName)
		spec, err := artifact.Get(artifactPath)
		if err != nil {
			return nil, errors.WithMessagef(err, "get artifact at path '%s' at hash '%s'", artifactPath, hash)
		}
		artifacts = append(artifacts, spec)
	}
	return artifacts, nil
}
