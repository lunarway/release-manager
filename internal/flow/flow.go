package flow

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/lunarway/release-manager/internal/grafana"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/slack"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
)

var (
	ErrUnknownEnvironment = errors.New("unknown environment")
)

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
	Dev     Environment `json:"dev,omitempty"`
	Staging Environment `json:"staging,omitempty"`
	Prod    Environment `json:"prod,omitempty"`
}

type FlowOptions struct {
	ConfigRepoURL     string
	ArtifactFileName  string
	Service           string
	Environment       string
	CommitterName     string
	CommitterEmail    string
	SSHPrivateKeyPath string
	SlackToken        string
	GrafanaAPIKey     string
	GrafanaUrl        string
	Branch            string
	ArtifactID        string
}

func Status(ctx context.Context, configRepoURL, artifactFileName, service, sshPrivateKeyPath string) (StatusResponse, error) {
	sourceConfigRepoPath, close, err := tempDir("k8s-config-status")
	if err != nil {
		return StatusResponse{}, err
	}
	defer close()
	// find current released artifact.json for each environment
	log.Debugf("Cloning source config repo %s into %s", configRepoURL, sourceConfigRepoPath)
	_, err = git.Clone(ctx, configRepoURL, sourceConfigRepoPath, sshPrivateKeyPath)
	if err != nil {
		return StatusResponse{}, errors.WithMessage(err, fmt.Sprintf("clone '%s' into '%s'", configRepoURL, sourceConfigRepoPath))
	}

	devSpec, err := envSpec(sourceConfigRepoPath, artifactFileName, service, "dev")
	if err != nil {
		cause := errors.Cause(err)
		if cause != artifact.ErrFileNotFound && cause != artifact.ErrNotParsable && cause != artifact.ErrUnknownFields {
			return StatusResponse{}, errors.WithMessage(err, "locate source spec for env dev")
		}
	}

	stagingSpec, err := envSpec(sourceConfigRepoPath, artifactFileName, service, "staging")
	if err != nil {
		cause := errors.Cause(err)
		if cause != artifact.ErrFileNotFound && cause != artifact.ErrNotParsable && cause != artifact.ErrUnknownFields {
			return StatusResponse{}, errors.WithMessage(err, "locate source spec for env staging")
		}
	}

	prodSpec, err := envSpec(sourceConfigRepoPath, artifactFileName, service, "prod")
	if err != nil {
		cause := errors.Cause(err)
		if cause != artifact.ErrFileNotFound && cause != artifact.ErrNotParsable && cause != artifact.ErrUnknownFields {
			return StatusResponse{}, errors.WithMessage(err, "locate source spec for env prod")
		}
	}

	return StatusResponse{
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

func envSpec(root, artifactFileName, service, env string) (artifact.Spec, error) {
	return artifact.Get(path.Join(releasePath(root, service, env), artifactFileName))
}

// sourceSpec returns the Spec of the current release.
func sourceSpec(root, artifactFileName, service, env string) (artifact.Spec, error) {
	var specPath string
	switch env {
	case "dev":
		specPath = path.Join(artifactPath(root, service, "master"), artifactFileName)
	case "staging":
		specPath = path.Join(releasePath(root, service, "dev"), artifactFileName)
	case "prod":
		specPath = path.Join(releasePath(root, service, "staging"), artifactFileName)
	default:
		return artifact.Spec{}, ErrUnknownEnvironment
	}
	log.Debugf("Get artifact spec from %s\n", specPath)
	return artifact.Get(specPath)
}

func srcPath(root, service, branch, env string) string {
	return path.Join(artifactPath(root, service, branch), env)
}

func artifactPath(root, service, branch string) string {
	return path.Join(root, "artifacts", service, branch)
}

func releasePath(root, service, env string) string {
	return path.Join(root, env, "releases", env, service)
}

// PushArtifact pushes an artifact into the configuration repository.
//
// The resourceRoot specifies the path to the artifact files. All files in this
// path will be pushed.
func PushArtifact(ctx context.Context, configRepoURL, artifactFileName, resourceRoot, sshPrivateKeyPath string) (string, error) {
	artifactSpecPath := path.Join(resourceRoot, artifactFileName)
	artifactSpec, err := artifact.Get(artifactSpecPath)
	if err != nil {
		return "", errors.WithMessagef(err, "path '%s'", artifactSpecPath)
	}
	artifactConfigRepoPath, close, err := tempDir("k8s-config-artifact")
	if err != nil {
		return "", err
	}
	defer close()
	// fmt.Printf is used for logging as this is called from artifact cli only
	fmt.Printf("Checkout config repository from '%s' into '%s'\n", configRepoURL, resourceRoot)
	repo, err := git.CloneDepth(context.Background(), configRepoURL, artifactConfigRepoPath, sshPrivateKeyPath, 1)
	if err != nil {
		return "", errors.WithMessage(err, "clone config repo")
	}
	destinationPath := artifactPath(artifactConfigRepoPath, artifactSpec.Service, artifactSpec.Application.Branch)
	fmt.Printf("Artifacts destination '%s'\n", destinationPath)
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
	committerName, committerEmail, err := git.CommitterDetails()
	if err != nil {
		return "", errors.WithMessage(err, "get committer details")
	}
	artifactID := artifactSpec.ID
	authorName := artifactSpec.Application.AuthorName
	authorEmail := artifactSpec.Application.AuthorEmail
	commitMsg := git.ArtifactCommitMessage(artifactSpec.Service, artifactID, authorName)
	fmt.Printf("Committing changes\n")
	err = git.Commit(context.Background(), repo, ".", authorName, authorEmail, committerName, committerEmail, commitMsg, sshPrivateKeyPath)
	if err != nil {
		if err == git.ErrNothingToCommit {
			return "", nil
		}
		return "", errors.WithMessage(err, "commit files")
	}
	return artifactSpec.ID, nil
}

type NotifyReleaseOptions struct {
	GrafanaBaseUrl string
	GrafanaApiKey  string
	SlackToken     string
	Environment    string
	Service        string
	ArtifactID     string
	Squad          string
	CommitAuthor   string
	CommitMessage  string
	CommitSHA      string
	CommitLink     string
	Releaser       string
}

func notifyRelease(opts NotifyReleaseOptions) error {
	client, err := slack.NewClient(opts.SlackToken)
	if err != nil {
		return err
	}
	err = client.NotifySlackReleasesChannel(slack.ReleaseOptions{
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

	err = grafana.Annotate(opts.GrafanaApiKey, opts.GrafanaBaseUrl, grafana.AnnotateRequest{
		What: fmt.Sprintf("Deployment: %s", opts.Service),
		Data: fmt.Sprintf("Author: %s\nMessage: %s\nArtifactID: %s", opts.CommitAuthor, opts.CommitMessage, opts.ArtifactID),
		Tags: []string{"deployment", opts.Service},
	})
	if err != nil {
		return err
	}

	log.WithFields("service", opts.Service,
		"environment", opts.Environment,
		"artifact-id", opts.ArtifactID,
		"commit-message", opts.CommitMessage,
		"commit-author", opts.CommitAuthor,
		"commit-link", opts.CommitLink,
		"commit-sha", opts.CommitSHA,
		"releaser", opts.Releaser,
		"type", "release").Infof("Release [%s]: %s (%s) by %s, author %s", opts.Environment, opts.Service, opts.ArtifactID, opts.Releaser, opts.CommitAuthor)

	return nil
}
