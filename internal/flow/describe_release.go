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
	"github.com/pkg/errors"
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

	var currentOffset uint = 0
	for count-int(currentOffset) > 0 {
		hash, err := s.Git.LocateServiceReleaseRollbackSkip(ctx, sourceRepo, environment, service, currentOffset)
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

		currentNamespace := ""

		r := regexp.MustCompile(fmt.Sprintf(`^(?P<environment>[^/]+)/releases/(?P<namespace>[^/]+)/(?P<service>[^/]+)/%s$`, regexp.QuoteMeta(s.ArtifactFileName)))

		gitFilesSpan, _ := s.Tracer.FromCtx(ctx, "finding namespace in git commit stats")
		gitFilesSpan.SetTag("gitcommit", commitObj.Hash.String())
		stats, err := commitObj.Stats()
		if err != nil {
			return DescribeReleaseResponse{}, errors.WithMessagef(err, "could not find commit stats for %s", commitObj.Hash.String())
		}
		for _, stat := range stats {
			match := r.FindStringSubmatch(stat.Name)
			if match != nil {
				currentNamespace = match[2]
				break
			}
		}
		gitFilesSpan.Finish()

		if currentNamespace == "" {
			return DescribeReleaseResponse{}, errors.Errorf("could not find namespace in commit '%s'", commitObj.Hash.String())
		}

		err = s.Git.Checkout(ctx, sourceConfigRepoPath, hash)
		if err != nil {
			return DescribeReleaseResponse{}, errors.WithMessagef(err, "checkout of commit %s", hash)
		}
		spec, err := envSpec(sourceConfigRepoPath, s.ArtifactFileName, service, environment, currentNamespace)
		if err != nil {
			return DescribeReleaseResponse{}, errors.WithMessagef(err, "reading artifact for commit %s", hash)
		}

		releases = append(releases, Release{
			Artifact:          spec,
			DefaultNamespaces: currentNamespace == environment,
			ReleaseIndex:      int(currentOffset),
			ReleasedAt:        commitObj.Committer.When,
			ReleasedByEmail:   commitInfo.ReleasedBy.Email,
			ReleasedByName:    commitInfo.ReleasedBy.Name,
			Intent:            commitInfo.Intent,
		})

		currentOffset++
	}

	return DescribeReleaseResponse{
		Releases: releases,
	}, nil
}
