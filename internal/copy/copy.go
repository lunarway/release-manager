package copy

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

var (
	// ErrUnknownSource indicates that the source directory was not found.
	ErrUnknownSource = errors.New("copy: unknown source")
)

func CopyDir(ctx context.Context, src, dest string) error {
	if !strings.HasSuffix(src, string(os.PathSeparator)) {
		src = fmt.Sprintf("%s/.", src)
	}
	return execCommand(ctx, ".", "cp", "-a", src, dest)
}

func CopyFile(ctx context.Context, src, dest string) error {
	return execCommand(ctx, ".", "cp", "-a", src, dest)
}

func execCommand(ctx context.Context, rootPath string, cmdName string, args ...string) error {
	logger := log.WithContext(ctx).WithFields("root", rootPath)
	logger.Infof("copy/execCommand: running: %s %s", cmdName, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, cmdName, args...)
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
	logger.Infof("copy/execCommand: exec command '%s %s': stdout: %s", cmdName, strings.Join(args, " "), stdoutData)
	logger.Infof("copy/execCommand: exec command '%s %s': stderr: %s", cmdName, strings.Join(args, " "), stderrData)
	if err != nil {
		match, regexpErr := regexp.Match("(?i)No such file or directory", stderrData)
		if regexpErr != nil {
			logger.Errorf("copy/execCommand: failed to detect if cp error is caused by unknown source: %v", regexpErr)
		}
		if match {
			return ErrUnknownSource
		}
		return errors.WithMessage(err, "execute command failed")
	}
	return nil
}
