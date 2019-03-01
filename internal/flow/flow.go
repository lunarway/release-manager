package flow

import (
	"fmt"
	"os"
	"path"

	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/spec"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
)

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
// .artifact.json spec for the service and previous environment.
// In this spec, find the image tag used as a key for locating the build.
//
// Checkout the Git hash of the kubernetes config repository from where the
// current release was based.
//
// Copy artifacts from the current release into the new environment and commit the changes
func Promote(configRepoURL, configRepoPath, artifactFileName, service, env string) error {
	// find current released .artifact.json for service in env - 1 (dev for staging, staging for prod)
	repo, err := git.Clone(configRepoURL, configRepoPath)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("clone '%s' into '%s'", configRepoURL, configRepoPath))
	}
	sourceSpec, err := sourceSpec(configRepoPath, artifactFileName, service, env)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("locate source spec"))
	}

	// find release identifier in .artifact.json
	pushStage, ok := sourceSpec.GetStage("push")
	if !ok {
		return errors.New("no push stage available for release")
	}
	pushData, ok := pushStage.Data.(map[string]interface{})
	if !ok {
		fmt.Printf("stage data type %[1]T data: %+[1]v\n", pushStage.Data)
		return errors.New("push stage data not of correct type: this should never happen")
	}
	release := pushData["tag"].(string)

	// ckechout commit of release
	hash, err := git.LocateRelease(configRepoURL, release)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("locate release '%s' from '%s'", release, configRepoURL))
	}
	err = git.Checkout(repo, hash)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("checkout '%s' hash '%s'", configRepoPath, hash))
	}

	// release service to env from original release
	sourcePath := srcPath(configRepoPath, service, env)
	destinationPath := releasePath(configRepoPath, service, env)
	fmt.Printf("source: %s\ndestin: %s\n", sourcePath, destinationPath)

	// empty existing resources in destination
	err = os.RemoveAll(destinationPath)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("remove destination path '%s'", destinationPath))
	}
	err = os.MkdirAll(destinationPath, os.ModePerm)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("create destination dir '%s'", destinationPath))
	}
	// copy previous env. files into destination
	err = copy.Copy(sourcePath, destinationPath)
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("copy resources from '%s' to '%s'", sourcePath, destinationPath))
	}

	// commit changes
	err = git.Commit(repo, releasePath(".", service, env), service, env, release, "BSO", "bso@lunarway.com")
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("commit changes from path '%s'", destinationPath))
	}

	return nil
}

// sourceSpec returns the Spec of the current release.
//
// env dev
// promote master branch to dev
//
// env staging
// promote staging config from commit of dev release
//
// env prod
// promote prod config from build commit of staging release
func sourceSpec(root, artifactFileName, service, env string) (spec.Spec, error) {
	switch env {
	case "dev":
		return spec.Get(path.Join(buildPath(root, service, "master"), artifactFileName))
	case "staging":
		return spec.Get(path.Join(releasePath(root, service, "dev"), artifactFileName))
	case "prod":
		return spec.Get(path.Join(releasePath(root, service, "staging"), artifactFileName))
	default:
		return spec.Spec{}, errors.New("unknown environment")
	}
}

func srcPath(root, service, env string) string {
	return path.Join(root, "builds", service, "master", env)

}

func buildPath(root, service, branch string) string {
	return path.Join(root, "builds", service, branch)
}

func releasePath(root, service, env string) string {
	return path.Join(root, env, "releases", env, service)
}
