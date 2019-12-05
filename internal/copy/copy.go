package copy

import (
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

func Copy(src, dest string) error {
	return execCommand(".", "cp", "-a", src, dest)
}

func execCommand(rootPath string, cmdName string, args ...string) error {
	log.WithFields("root", rootPath).Infof("copy/execCommand: running: %s %s", cmdName, strings.Join(args, " "))
	cmd := exec.Command(cmdName, args...)
	cmd.Dir = rootPath
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.WithMessage(err, "get stdout pipe for command")
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errors.WithMessage(err, "get stderr pipe for command")
	}
	err = cmd.Start()
	if err != nil {
		return errors.WithMessage(err, "start command")
	}

	stdoutData, err := ioutil.ReadAll(stdout)
	if err != nil {
		return errors.WithMessage(err, "read stdout data of command")
	}
	stderrData, err := ioutil.ReadAll(stderr)
	if err != nil {
		return errors.WithMessage(err, "read stderr data of command")
	}

	err = cmd.Wait()
	log.Infof("copy/execCommand: exec command '%s %s': stdout: %s", cmdName, strings.Join(args, " "), stdoutData)
	log.Infof("copy/execCommand: exec command '%s %s': stderr: %s", cmdName, strings.Join(args, " "), stderrData)
	if err != nil {
		return errors.WithMessage(err, "execute command failed")
	}
	return nil
}
