package flow

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/commitinfo"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/intent"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type DescribeReleaseResponse struct {
	Releases []Release
}

type Release struct {
	DefaultNamespaces bool
	ReleaseIndex      int
	Artifact          artifact.Spec
	ReleasedAt        time.Time
	ReleasedByEmail   string
	ReleasedByName    string
	Intent            intent.Intent
}

// DescribeRelease returns information about a specific release in an environment.
func (s *Service) DescribeRelease(ctx context.Context, environment, service string, count int) (DescribeReleaseResponse, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.DescribeRelease")
	span.SetBaggageItem("count", fmt.Sprintf("%v", count))

	defer span.Finish()
	sourceConfigRepoPath, close, err := git.TempDirAsync(ctx, s.Tracer, "k8s-config-describe-release")
	if err != nil {
		return DescribeReleaseResponse{}, err
	}
	defer close(ctx)

	log.WithContext(ctx).Debugf("Cloning source config repo %s into %s", s.Git.ConfigRepoURL, sourceConfigRepoPath)
	sourceRepo, err := s.Git.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		return DescribeReleaseResponse{}, errors.WithMessagef(err, "clone into '%s'", sourceConfigRepoPath)
	}

	var releases []Release

	for currentOffset := 0; currentOffset < count; currentOffset++ {
		hash, err := s.Git.LocateServiceReleaseRollbackSkip(ctx, sourceRepo, environment, service, uint(currentOffset))
		if err != nil {
			if errors.Is(err, git.ErrReleaseNotFound) {
				break
			}
			return DescribeReleaseResponse{}, errors.WithMessagef(err, "locate release")
		}

		commitObj, err := sourceRepo.CommitObject(hash)
		if err != nil {
			return DescribeReleaseResponse{}, errors.WithMessagef(err, "get commit at hash '%s'", hash)
		}

		commitInfo, err := commitinfo.ParseCommitInfo(commitObj.Message)
		if err != nil {
			return DescribeReleaseResponse{}, errors.WithMessagef(err, "parse commit info at hash '%s'", hash)
		}

		namespace, err := findNamespaceFromCommit(ctx, commitObj, s.ArtifactFileName)
		if err != nil {
			return DescribeReleaseResponse{}, errors.WithMessagef(err, "could not find namespace for %s", commitObj.Hash.String())
		}

		err = s.Git.Checkout(ctx, sourceConfigRepoPath, hash)
		if err != nil {
			return DescribeReleaseResponse{}, errors.WithMessagef(err, "checkout of commit %s", hash)
		}
		spec, err := envSpec(sourceConfigRepoPath, s.ArtifactFileName, service, environment, namespace)
		if err != nil {
			return DescribeReleaseResponse{}, errors.WithMessagef(err, "reading artifact for commit %s", hash)
		}

		releases = append(releases, Release{
			Artifact:          spec,
			DefaultNamespaces: namespace == environment,
			ReleaseIndex:      int(currentOffset),
			ReleasedAt:        commitObj.Committer.When,
			ReleasedByEmail:   commitInfo.ReleasedBy.Email,
			ReleasedByName:    commitInfo.ReleasedBy.Name,
			Intent:            commitInfo.Intent,
		})
	}

	return DescribeReleaseResponse{
		Releases: releases,
	}, nil
}

func findNamespaceFromCommit(ctx context.Context, commitObj *object.Commit, artifactFileName string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "flow.findNamespace")
	defer span.Finish()
	span.SetTag("gitcommit", commitObj.Hash.String())

	r := regexp.MustCompile(fmt.Sprintf(`^(?P<environment>[^/]+)/releases/(?P<namespace>[^/]+)/(?P<service>[^/]+)/%s$`, regexp.QuoteMeta(artifactFileName)))
	stats, err := commitObj.Stats()
	if err != nil {
		return "", errors.WithMessagef(err, "could not find commit stats")
	}
	for _, stat := range stats {
		match := r.FindStringSubmatch(stat.Name)
		if match != nil {
			return match[2], nil
		}
	}

	return "", errors.Errorf("could not find namespace")
}
