package git

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

type LocalGitConfigAPI struct {
}

// NewLocalGitConfigAPI provide a GitConfigAPI for a local repository
func NewLocalGitConfigAPI() *LocalGitConfigAPI {
	return &LocalGitConfigAPI{}
}

// CommitterDetails returns name and email read for a Git configuration file.
//
// Fetching the configuration values are delegated to the git CLI and follows
// precedence rules defined by Git.
func (*LocalGitConfigAPI) CommitterDetails() (name string, email string, err error) {
	name, err = getValue("user.name", "HAMCTL_USER_NAME")
	if err != nil {
		return "", "", err
	}
	email, err = getValue("user.email", "HAMCTL_USER_EMAIL")
	if err != nil {
		return "", "", err
	}
	return name, email, nil
}

func getValue(gitKey, envKey string) (string, error) {
	v, ok := os.LookupEnv(envKey)
	if ok {
		return v, nil
	}
	v, err := getGitConfig(gitKey)
	if err != nil {
		return "", errors.WithMessagef(err, "Failed to get credentials with 'git config --get %s'", gitKey)
	}
	return v, nil
}

// getGitConfig reads a git configuration field and returns its value as a
// string.
func getGitConfig(field string) (string, error) {
	cmd := exec.Command("git", "config", "--get", field)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", errors.WithMessage(err, "get stdout pipe for command")
	}
	err = cmd.Start()
	if err != nil {
		return "", errors.WithMessage(err, "start command")
	}
	stdoutData, err := io.ReadAll(stdout)
	if err != nil {
		return "", errors.WithMessage(err, "read stdout data of command")
	}

	err = cmd.Wait()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(stdoutData)), nil
}
