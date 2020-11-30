package git

import (
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

// CommitterDetails returns name and email read for a Git configuration file.
//
// Fetching the configuration values are delegated to the git CLI and follows
// precedence rules defined by Git.
func CommitterDetails() (string, string, error) {
	name, err := getGitConfig("user.name")
	if err != nil {
		return "", "", errors.WithMessage(err, "Failed to get credentials with 'git config --get user.name'")
	}
	email, err := getGitConfig("user.email")
	if err != nil {
		return "", "", errors.WithMessage(err, "Failed to get credentials with 'git config --get user.email'")
	}
	return name, email, nil
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
	stdoutData, err := ioutil.ReadAll(stdout)
	if err != nil {
		return "", errors.WithMessage(err, "read stdout data of command")
	}

	err = cmd.Wait()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(stdoutData)), nil
}
