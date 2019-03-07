package flow

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/spec"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
)

var (
	sourceConfigRepoPath      = path.Join(".tmp", "k8s-config-source")
	destinationConfigRepoPath = path.Join(".tmp", "k8s-config-destination")
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

func Status(configRepoURL, artifactFileName, service string) (StatusResponse, error) {
	// find current released artifact.json for each environment
	fmt.Printf("Cloning source config repo %s into %s\n", configRepoURL, sourceConfigRepoPath)
	_, err := git.Clone(configRepoURL, sourceConfigRepoPath)
	if err != nil {
		return StatusResponse{}, errors.WithMessage(err, fmt.Sprintf("clone '%s' into '%s'", configRepoURL, sourceConfigRepoPath))
	}

	devSpec, err := envSpec(sourceConfigRepoPath, artifactFileName, service, "dev")
	if err != nil {
		return StatusResponse{}, errors.WithMessage(err, "locate source spec for env dev")
	}

	stagingSpec, err := envSpec(sourceConfigRepoPath, artifactFileName, service, "staging")
	if err != nil {
		return StatusResponse{}, errors.WithMessage(err, "locate source spec for env staging")
	}

	prodSpec, err := envSpec(sourceConfigRepoPath, artifactFileName, service, "prod")
	if err != nil {
		return StatusResponse{}, errors.WithMessage(err, "locate source spec for env prod")
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

func calculateTotalVulnerabilties(severity string, s spec.Spec) int64 {
	var result int64
	// for _, stage := range s.Stages {
	// 	if stage.ID == "snyk-code" {
	// 		result += stage.Data.
	// 	}
	// 	if stage.ID == "snyk-docker" {

	// 	}
	// }
	return result
}

// Promote promotes a specific service to environment env.
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
// Use the artifact ID as a key for locating the build.
//
// Find the commit with the artifact ID and checkout the config repository at
// this point.
//
// Copy artifacts from the current release into the new environment and commit
// the changes
func Promote(configRepoURL, artifactFileName, service, env string) (string, error) {
	// find current released artifact.json for service in env - 1 (dev for staging, staging for prod)
	fmt.Printf("Cloning source config repo %s into %s\n", configRepoURL, sourceConfigRepoPath)
	sourceRepo, err := git.Clone(configRepoURL, sourceConfigRepoPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("clone '%s' into '%s'", configRepoURL, sourceConfigRepoPath))
	}
	sourceSpec, err := sourceSpec(sourceConfigRepoPath, artifactFileName, service, env)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("locate source spec"))
	}

	// find release identifier in artifact.json
	release := sourceSpec.ID
	fmt.Printf("Found artifact id '%s'\n", release)

	// ckechout commit of release
	hash, err := git.LocateRelease(sourceRepo, release)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("locate release '%s' from '%s'", release, configRepoURL))
	}
	err = git.Checkout(sourceRepo, hash)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("checkout release hash '%s' from '%s'", hash, configRepoURL))
	}

	destinationRepo, err := git.Clone(configRepoURL, destinationConfigRepoPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("clone destination repo '%s' into '%s'", configRepoURL, destinationConfigRepoPath))
	}

	// release service to env from original release
	sourcePath := srcPath(sourceConfigRepoPath, service, env)
	destinationPath := releasePath(destinationConfigRepoPath, service, env)
	fmt.Printf("Copy resources from: %s\n", sourcePath)
	fmt.Printf("To:                  %s\n", destinationPath)

	// empty existing resources in destination
	err = os.RemoveAll(destinationPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("remove destination path '%s'", destinationPath))
	}
	err = os.MkdirAll(destinationPath, os.ModePerm)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("create destination dir '%s'", destinationPath))
	}
	// copy previous env. files into destination
	err = copy.Copy(sourcePath, destinationPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("copy resources from '%s' to '%s'", sourcePath, destinationPath))
	}
	// copy artifact spec
	artifactSourcePath := srcPath(sourceConfigRepoPath, service, artifactFileName)
	artifactDestinationPath := path.Join(releasePath(destinationConfigRepoPath, service, env), artifactFileName)
	fmt.Printf("Copy artifact from: %s\n", artifactSourcePath)
	fmt.Printf("To:                 %s\n", artifactDestinationPath)
	err = copy.Copy(artifactSourcePath, artifactDestinationPath)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("copy artifact spec from '%s' to '%s'", artifactSourcePath, artifactDestinationPath))
	}

	// commit changes
	committerName, committerEmail, err := committerDetails()
	if err != nil {
		return "", err
	}
	authorName := sourceSpec.Application.AuthorName
	authorEmail := sourceSpec.Application.AuthorEmail
	releaseMessage := fmt.Sprintf("[%s/%s] release %s", env, service, release)
	fmt.Printf("Committing release: %s\n", releaseMessage)
	fmt.Printf("  Author:    %s <%s>\n", authorName, authorEmail)
	fmt.Printf("  Committer: %s <%s>\n", committerName, committerEmail)
	err = git.Commit(destinationRepo, releasePath(".", service, env), authorName, authorEmail, committerName, committerEmail, releaseMessage)
	if err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("commit changes from path '%s'", destinationPath))
	}

	return release, nil
}

func committerDetails() (string, string, error) {
	c, err := git.GlobalConfig()
	if err != nil {
		return "", "", errors.WithMessage(err, "get global config")
	}
	committerName := c.Section("user").Option("name")
	committerEmail := c.Section("user").Option("email")
	if committerEmail == "" {
		return "", "", errors.New("user.email not available in global git config")
	}
	if committerName == "" {
		return "", "", errors.New("user.name not available in global git config")
	}
	return committerName, committerEmail, nil
}

func envSpec(root, artifactFileName, service, env string) (spec.Spec, error) {
	return spec.Get(path.Join(releasePath(root, service, env), artifactFileName))
}

// sourceSpec returns the Spec of the current release.
func sourceSpec(root, artifactFileName, service, env string) (spec.Spec, error) {
	var specPath string
	switch env {
	case "dev":
		specPath = path.Join(buildPath(root, service, "master"), artifactFileName)
	case "staging":
		specPath = path.Join(releasePath(root, service, "dev"), artifactFileName)
	case "prod":
		specPath = path.Join(releasePath(root, service, "staging"), artifactFileName)
	default:
		return spec.Spec{}, errors.New("unknown environment")
	}
	fmt.Printf("Get artifact spec from %s\n", specPath)
	return spec.Get(specPath)
}

func srcPath(root, service, env string) string {
	return path.Join(buildPath(root, service, "master"), env)
}

func buildPath(root, service, branch string) string {
	return path.Join(root, "builds", service, branch)
}

func releasePath(root, service, env string) string {
	return path.Join(root, env, "releases", env, service)
}
