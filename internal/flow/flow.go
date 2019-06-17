package flow

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/grafana"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/lunarway/release-manager/internal/try"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
)

var (
	ErrUnknownEnvironment            = errors.New("unknown environment")
	ErrNamespaceNotAllowedByArtifact = errors.New("namespace not allowed by artifact")
)

type Service struct {
	ArtifactFileName string
	UserMappings     map[string]string
	Slack            *slack.Client
	Grafana          *grafana.Service
	Git              *git.Service

	MaxRetries int
}

// retry tries the function f until max attempts is reached
// If f returns a true bool or a nil error retries are stopped and the error is
// returned.
func (s *Service) retry(ctx context.Context, f func(int) (bool, error)) error {
	return try.Do(ctx, s.MaxRetries, f)
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
	sourceConfigRepoPath, close, err := git.TempDir("k8s-config-status")
	if err != nil {
		return StatusResponse{}, err
	}
	defer close()
	// find current released artifact.json for each environment
	log.Debugf("Cloning source config repo %s into %s", s.Git.ConfigRepoURL, sourceConfigRepoPath)
	_, err = s.Git.Clone(ctx, sourceConfigRepoPath)
	if err != nil {
		return StatusResponse{}, errors.WithMessage(err, fmt.Sprintf("clone into '%s'", sourceConfigRepoPath))
	}

	defaultNamespaces := namespace == ""
	defaultNamespace := func(env string) string {
		if defaultNamespaces {
			return env
		}
		return namespace
	}
	devSpec, err := envSpec(sourceConfigRepoPath, s.ArtifactFileName, service, "dev", defaultNamespace("dev"))
	if err != nil {
		cause := errors.Cause(err)
		if cause != artifact.ErrFileNotFound && cause != artifact.ErrNotParsable && cause != artifact.ErrUnknownFields {
			return StatusResponse{}, errors.WithMessage(err, "locate source spec for env dev")
		}
	}

	stagingSpec, err := envSpec(sourceConfigRepoPath, s.ArtifactFileName, service, "staging", defaultNamespace("staging"))
	if err != nil {
		cause := errors.Cause(err)
		if cause != artifact.ErrFileNotFound && cause != artifact.ErrNotParsable && cause != artifact.ErrUnknownFields {
			return StatusResponse{}, errors.WithMessage(err, "locate source spec for env staging")
		}
	}

	prodSpec, err := envSpec(sourceConfigRepoPath, s.ArtifactFileName, service, "prod", defaultNamespace("prod"))
	if err != nil {
		cause := errors.Cause(err)
		if cause != artifact.ErrFileNotFound && cause != artifact.ErrNotParsable && cause != artifact.ErrUnknownFields {
			return StatusResponse{}, errors.WithMessage(err, "locate source spec for env prod")
		}
	}

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
func sourceSpec(root, artifactFileName, service, env, namespace string) (artifact.Spec, error) {
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
	log.Infof("Get artifact spec from %s", specPath)
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
	artifactConfigRepoPath, close, err := git.TempDir("k8s-config-artifact")
	if err != nil {
		return "", err
	}
	defer close()
	// fmt.Printf is used for logging as this is called from artifact cli only
	fmt.Printf("Checkout config repository from '%s' into '%s'\n", gitSvc.ConfigRepoURL, resourceRoot)
	listFiles(resourceRoot)
	repo, err := gitSvc.Clone(context.Background(), artifactConfigRepoPath)
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
	err = copy.Copy(resourceRoot, destinationPath)
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
	err = gitSvc.Commit(context.Background(), repo, ".", authorName, authorEmail, committerName, committerEmail, commitMsg)
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

type NotifyReleaseOptions struct {
	Environment   string
	Namespace     string
	Service       string
	ArtifactID    string
	Squad         string
	CommitAuthor  string
	CommitMessage string
	CommitSHA     string
	CommitLink    string
	Releaser      string
}

func (s *Service) notifyRelease(opts NotifyReleaseOptions) error {
	err := s.Slack.NotifySlackReleasesChannel(slack.ReleaseOptions{
		Service:       opts.Service,
		Environment:   opts.Environment,
		ArtifactID:    opts.ArtifactID,
		CommitMessage: opts.CommitMessage,
		CommitAuthor:  opts.CommitAuthor,
		CommitLink:    opts.CommitLink,
		CommitSHA:     opts.CommitSHA,
		Releaser:      opts.Releaser,
	})
	if err != nil {
		return err
	}

	err = s.Grafana.Annotate(opts.Environment, grafana.AnnotateRequest{
		What: fmt.Sprintf("Deployment: %s", opts.Service),
		Data: fmt.Sprintf("Author: %s\nMessage: %s\nArtifactID: %s", opts.CommitAuthor, opts.CommitMessage, opts.ArtifactID),
		Tags: []string{"deployment", opts.Service},
	})
	if err != nil {
		return err
	}

	log.WithFields("service", opts.Service,
		"environment", opts.Environment,
		"namespace", opts.Namespace,
		"artifact-id", opts.ArtifactID,
		"commit-message", opts.CommitMessage,
		"commit-author", opts.CommitAuthor,
		"commit-link", opts.CommitLink,
		"commit-sha", opts.CommitSHA,
		"releaser", opts.Releaser,
		"type", "release").Infof("Release [%s]: %s (%s) by %s, author %s", opts.Environment, opts.Service, opts.ArtifactID, opts.Releaser, opts.CommitAuthor)

	return nil
}
