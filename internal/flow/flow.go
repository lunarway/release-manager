package flow

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/copy"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/lunarway/release-manager/internal/try"
	"github.com/pkg/errors"
)

var (
	ErrUnknownEnvironment            = errors.New("unknown environment")
	ErrNamespaceNotAllowedByArtifact = errors.New("namespace not allowed by artifact")
	ErrUnknownConfiguration          = errors.New("unknown configuration")
)

type Service struct {
	ArtifactFileName string
	UserMappings     map[string]string
	Slack            *slack.Client
	Git              *git.Service
	Tracer           tracing.Tracer

	PublishPromote           func(context.Context, PromoteEvent) error
	PublishRollback          func(context.Context, RollbackEvent) error
	PublishReleaseArtifactID func(context.Context, ReleaseArtifactIDEvent) error
	PublishReleaseBranch     func(context.Context, ReleaseBranchEvent) error

	MaxRetries int

	// NotifyReleaseHook is triggered in a Go routine when a release is completed.
	// The context.Context is cancelled if the originating flow call is cancelled.
	NotifyReleaseHook func(ctx context.Context, options NotifyReleaseOptions)
}

type NotifyReleaseOptions struct {
	Environment string
	Namespace   string
	Service     string
	Releaser    string
	Spec        artifact.Spec
}

// retry tries the function f until max attempts is reached
// If f returns a true bool or a nil error retries are stopped and the error is
// returned.
func (s *Service) retry(ctx context.Context, f func(context.Context, int) (bool, error)) error {
	return try.Do(ctx, s.Tracer, s.MaxRetries, func(ctx context.Context, attempt int) (bool, error) {
		stop, err := f(ctx, attempt)
		if err != nil {
			if errors.Cause(err) == git.ErrBranchBehindOrigin {
				log.WithContext(ctx).Infof("flow/retry: master repo not aligned with origin. Syncing and retrying")
				err := s.Git.SyncMaster(ctx)
				if err != nil {
					return false, errors.WithMessage(err, "sync master")
				}
			}
		}
		return stop, err
	})
}

type Environment struct {
	Tag                   string    `json:"tag,omitempty"`
	Committer             string    `json:"committer,omitempty"`
	Author                string    `json:"author,omitempty"`
	Message               string    `json:"message,omitempty"`
	Date                  time.Time `json:"date,omitempty"`
	BuildURL              string    `json:"buildUrl,omitempty"`
	HighVulnerabilities   int64     `json:"highVulnerabilities,omitempty"`
	MediumVulnerabilities int64     `json:"mediumVulnerabilities,omitempty"`
	LowVulnerabilities    int64     `json:"lowVulnerabilities,omitempty"`
}

type StatusResponse struct {
	DefaultNamespaces bool        `json:"defaultNamespaces,omitempty"`
	Dev               Environment `json:"dev,omitempty"`
	Staging           Environment `json:"staging,omitempty"`
	Prod              Environment `json:"prod,omitempty"`
}

type Actor struct {
	Email string
	Name  string
}

func (s *Service) Status(ctx context.Context, namespace, service string) (StatusResponse, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.Status")
	defer span.Finish()

	defaultNamespaces := namespace == ""
	defaultNamespace := func(env string) string {
		if defaultNamespaces {
			return env
		}
		return namespace
	}
	span, _ = s.Tracer.FromCtx(ctx, "artifact spec for environment")
	span.SetTag("env", "dev")
	devSpec, err := envSpec(s.Git.MasterPath, s.ArtifactFileName, service, "dev", defaultNamespace("dev"))
	if err != nil {
		cause := errors.Cause(err)
		if cause != artifact.ErrFileNotFound && cause != artifact.ErrNotParsable && cause != artifact.ErrUnknownFields {
			return StatusResponse{}, errors.WithMessage(err, "locate source spec for env dev")
		}
	}
	defer span.Finish()

	span, _ = s.Tracer.FromCtx(ctx, "artifact spec for environment")
	span.SetTag("env", "staging")
	stagingSpec, err := envSpec(s.Git.MasterPath, s.ArtifactFileName, service, "staging", defaultNamespace("staging"))
	if err != nil {
		cause := errors.Cause(err)
		if cause != artifact.ErrFileNotFound && cause != artifact.ErrNotParsable && cause != artifact.ErrUnknownFields {
			return StatusResponse{}, errors.WithMessage(err, "locate source spec for env staging")
		}
	}
	defer span.Finish()

	span, _ = s.Tracer.FromCtx(ctx, "artifact spec for environment")
	span.SetTag("env", "prod")
	prodSpec, err := envSpec(s.Git.MasterPath, s.ArtifactFileName, service, "prod", defaultNamespace("prod"))
	if err != nil {
		cause := errors.Cause(err)
		if cause != artifact.ErrFileNotFound && cause != artifact.ErrNotParsable && cause != artifact.ErrUnknownFields {
			return StatusResponse{}, errors.WithMessage(err, "locate source spec for env prod")
		}
	}
	defer span.Finish()

	return StatusResponse{
		DefaultNamespaces: defaultNamespaces,
		Dev: Environment{
			Tag:                   devSpec.ID,
			Committer:             devSpec.Application.CommitterName,
			Author:                devSpec.Application.AuthorName,
			Message:               devSpec.Application.Message,
			Date:                  devSpec.CI.End,
			BuildURL:              devSpec.CI.JobURL,
			HighVulnerabilities:   calculateTotalVulnerabilties("high", devSpec),
			MediumVulnerabilities: calculateTotalVulnerabilties("medium", devSpec),
			LowVulnerabilities:    calculateTotalVulnerabilties("low", devSpec),
		},
		Staging: Environment{
			Tag:                   stagingSpec.ID,
			Committer:             stagingSpec.Application.CommitterName,
			Author:                stagingSpec.Application.AuthorName,
			Message:               stagingSpec.Application.Message,
			Date:                  stagingSpec.CI.End,
			BuildURL:              stagingSpec.CI.JobURL,
			HighVulnerabilities:   calculateTotalVulnerabilties("high", stagingSpec),
			MediumVulnerabilities: calculateTotalVulnerabilties("medium", stagingSpec),
			LowVulnerabilities:    calculateTotalVulnerabilties("low", stagingSpec),
		},
		Prod: Environment{
			Tag:                   prodSpec.ID,
			Committer:             prodSpec.Application.CommitterName,
			Author:                prodSpec.Application.AuthorName,
			Message:               prodSpec.Application.Message,
			Date:                  prodSpec.CI.End,
			BuildURL:              prodSpec.CI.JobURL,
			HighVulnerabilities:   calculateTotalVulnerabilties("high", prodSpec),
			MediumVulnerabilities: calculateTotalVulnerabilties("medium", prodSpec),
			LowVulnerabilities:    calculateTotalVulnerabilties("low", prodSpec),
		},
	}, nil
}

func calculateTotalVulnerabilties(severity string, s artifact.Spec) int64 {
	result := float64(0)
	for _, stage := range s.Stages {
		if stage.ID == "snyk-code" {
			data := stage.Data.(map[string]interface{})
			vulnerabilities := data["vulnerabilities"].(map[string]interface{})
			result += vulnerabilities[severity].(float64)
		}
		if stage.ID == "snyk-docker" {
			data := stage.Data.(map[string]interface{})
			vulnerabilities := data["vulnerabilities"].(map[string]interface{})
			result += vulnerabilities[severity].(float64)
		}
	}
	return int64(result + 0.5)
}

func envSpec(root, artifactFileName, service, env, namespace string) (artifact.Spec, error) {
	return artifact.Get(path.Join(releasePath(root, service, env, namespace), artifactFileName))
}

// sourceSpec returns the Spec of the current release.
func sourceSpec(ctx context.Context, root, artifactFileName, service, env, namespace string) (artifact.Spec, error) {
	var specPath string
	switch env {
	case "dev":
		specPath = path.Join(artifactPath(root, service, "master"), artifactFileName)
	case "staging":
		// if namespace is set to the environment we have to look one environment back when locating the artifact.json
		if namespace == "staging" {
			namespace = "dev"
		}
		specPath = path.Join(releasePath(root, service, "dev", namespace), artifactFileName)
	case "prod":
		// if namespace is set to the environment we have to look one environment back when locating the artifact.json
		if namespace == "prod" {
			namespace = "staging"
		}
		specPath = path.Join(releasePath(root, service, "staging", namespace), artifactFileName)
	default:
		return artifact.Spec{}, ErrUnknownEnvironment
	}
	log.WithContext(ctx).Infof("Get artifact spec from %s", specPath)
	return artifact.Get(specPath)
}

func srcPath(root, service, branch, env string) string {
	return path.Join(artifactPath(root, service, branch), env)
}

func artifactPath(root, service, branch string) string {
	return path.Join(root, "artifacts", service, branch)
}

func releasePath(root, service, env, namespace string) string {
	return path.Join(root, env, "releases", namespace, service)
}

// PushArtifact pushes an artifact into the configuration repository.
//
// The resourceRoot specifies the path to the artifact files. All files in this
// path will be pushed.
func PushArtifact(ctx context.Context, gitSvc *git.Service, artifactFileName, resourceRoot string) (string, error) {
	artifactSpecPath := path.Join(resourceRoot, artifactFileName)
	artifactSpec, err := artifact.Get(artifactSpecPath)
	if err != nil {
		return "", errors.WithMessagef(err, "path '%s'", artifactSpecPath)
	}
	artifactConfigRepoPath, close, err := git.TempDir(ctx, gitSvc.Tracer, "k8s-config-artifact")
	if err != nil {
		return "", err
	}
	defer close(ctx)
	// fmt.Printf is used for logging as this is called from artifact cli only
	fmt.Printf("Checkout config repository from '%s' into '%s'\n", gitSvc.ConfigRepoURL, resourceRoot)
	listFiles(resourceRoot)
	_, err = gitSvc.Clone(ctx, artifactConfigRepoPath)
	if err != nil {
		return "", errors.WithMessage(err, "clone config repo")
	}
	destinationPath := artifactPath(artifactConfigRepoPath, artifactSpec.Service, artifactSpec.Application.Branch)
	fmt.Printf("Artifacts destination '%s'\n", destinationPath)
	listFiles(destinationPath)
	fmt.Printf("Removing existing files\n")
	err = os.RemoveAll(destinationPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("remove destination path '%s'", destinationPath))
	}
	err = os.MkdirAll(destinationPath, os.ModePerm)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("create destination dir '%s'", destinationPath))
	}
	fmt.Printf("Copy configuration into destination\n")
	err = copy.CopyDir(ctx, resourceRoot, destinationPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("copy resources from '%s' to '%s'", resourceRoot, destinationPath))
	}
	listFiles(destinationPath)
	committerName, committerEmail, err := git.CommitterDetails()
	if err != nil {
		return "", errors.WithMessage(err, "get committer details")
	}
	artifactID := artifactSpec.ID
	authorName := artifactSpec.Application.AuthorName
	authorEmail := artifactSpec.Application.AuthorEmail
	commitMsg := git.ArtifactCommitMessage(artifactSpec.Service, artifactID, authorName)
	fmt.Printf("Committing changes\n")
	err = gitSvc.Commit(ctx, destinationPath, ".", authorName, authorEmail, committerName, committerEmail, commitMsg)
	if err != nil {
		if err == git.ErrNothingToCommit {
			return artifactSpec.ID, nil
		}
		return "", errors.WithMessage(err, "commit files")
	}
	return artifactSpec.ID, nil
}

func listFiles(path string) {
	fmt.Printf("Files in path '%s'\n", path)
	err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("failed to walk dir: %v\n", err)
				return nil
			}
			fmt.Printf("  %s\n", path)
			return nil
		})
	if err != nil {
		fmt.Printf("failed to read dir: %v\n", err)
	}
}

func (s *Service) notifyRelease(ctx context.Context, opts NotifyReleaseOptions) {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.notifyRelease")
	defer span.Finish()
	if s.NotifyReleaseHook != nil {
		go s.NotifyReleaseHook(noCancel{ctx: ctx}, opts)
	}
}

// noCancel is a context.Context that does not propagate cancellations.
type noCancel struct {
	ctx context.Context
}

func (c noCancel) Deadline() (time.Time, bool)       { return time.Time{}, false }
func (c noCancel) Done() <-chan struct{}             { return nil }
func (c noCancel) Err() error                        { return nil }
func (c noCancel) Value(key interface{}) interface{} { return c.ctx.Value(key) }

func (s *Service) cleanCopy(ctx context.Context, src, dest string) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.cleanCopy")
	defer span.Finish()
	span, _ = s.Tracer.FromCtx(ctx, "remove destination")
	err := os.RemoveAll(dest)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "remove destination path")
	}
	span, _ = s.Tracer.FromCtx(ctx, "create destination dir")
	err = os.MkdirAll(dest, os.ModePerm)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "create destination dir")
	}
	span, _ = s.Tracer.FromCtx(ctx, "copy files")
	span.Finish()
	err = copy.CopyDir(ctx, src, dest)
	if err != nil {
		if errors.Cause(err) == copy.ErrUnknownSource {
			return ErrUnknownConfiguration
		}
		return errors.WithMessage(err, "copy files")
	}
	return nil
}
