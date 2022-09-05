package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/lunarway/release-manager/internal/commitinfo"
	"github.com/lunarway/release-manager/internal/copy"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/pkg/errors"
)

var (
	ErrNothingToCommit    = errors.New("nothing to commit")
	ErrReleaseNotFound    = errors.New("release not found")
	ErrBranchBehindOrigin = errors.New("branch behind origin")
	ErrUnknownGit         = errors.New("unknown git error")
)

type GitConfig struct {
	User       string
	Email      string
	SigningKey string
}

type Service struct {
	Tracer            tracing.Tracer
	Copier            *copy.Copier
	SSHPrivateKeyPath string
	ConfigRepoURL     string
	Config            *GitConfig
	ArtifactFileName  string

	masterPath  string
	masterMutex sync.RWMutex
	master      *git.Repository
}

func (s *Service) MasterPath() string {
	return s.masterPath
}

// InitMasterRepo clones the configuration repository into a master directory.
func (s *Service) InitMasterRepo(ctx context.Context) (func(context.Context), error) {
	span, ctx := s.Tracer.FromCtx(ctx, "git.InitMasterRepo")
	defer span.Finish()
	path, close, err := TempDir(ctx, s.Tracer, "k8s-master-clone")
	if err != nil {
		close(ctx)
		return nil, errors.WithMessage(err, "get temporary directory")
	}
	repo, err := s.clone(ctx, path)
	if err != nil {
		close(ctx)
		return nil, errors.WithMessagef(err, "clone into '%s'", path)
	}
	s.masterMutex.Lock()
	defer s.masterMutex.Unlock()
	s.master = repo
	s.masterPath = path
	log.WithContext(ctx).Infof("Master repo cloned into '%s'", path)
	return close, nil
}

func (s *Service) clone(ctx context.Context, destination string) (*git.Repository, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "git.clone")
	defer span.Finish()
	authSSH, err := ssh.NewPublicKeysFromFile("git", s.SSHPrivateKeyPath, "")
	if err != nil {
		return nil, errors.WithMessage(err, "public keys from file")
	}
	span, _ = s.Tracer.FromCtx(ctx, "remove destination")
	span.SetTag("path", destination)
	err = os.RemoveAll(destination)
	span.Finish()
	if err != nil {
		return nil, errors.WithMessage(err, "remove existing destination")
	}

	span, _ = s.Tracer.FromCtx(ctx, "plain clone")
	r, err := git.PlainCloneContext(ctx, destination, false, &git.CloneOptions{
		URL:  s.ConfigRepoURL,
		Auth: authSSH,
	})
	span.Finish()
	if err != nil {
		return nil, errors.WithMessage(err, "clone repo")
	}
	return r, nil
}

// SyncMaster pulls latest changes from master repo.
func (s *Service) SyncMaster(ctx context.Context) error {
	span, ctx := s.Tracer.FromCtx(ctx, "git.SyncMaster")
	defer span.Finish()
	span, _ = s.Tracer.FromCtx(ctx, "lock mutex")
	s.masterMutex.Lock()
	defer s.masterMutex.Unlock()
	span.Finish()

	span, _ = s.Tracer.FromCtx(ctx, "fetch")
	err := execCommand(ctx, s.MasterPath(), "git", "fetch", "origin", "master")
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "fetch changes")
	}
	span, _ = s.Tracer.FromCtx(ctx, "pull")
	err = execCommand(ctx, s.MasterPath(), "git", "pull")
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "pull latest")
	}
	return nil
}

// Clone returns a Git repository copy from the master repository.
func (s *Service) Clone(ctx context.Context, destination string) (*git.Repository, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "git.Clone")
	defer span.Finish()
	return s.copyMaster(ctx, destination)
}

func (s *Service) copyMaster(ctx context.Context, destination string) (*git.Repository, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "git.copyMaster")
	defer span.Finish()
	span, _ = s.Tracer.FromCtx(ctx, "remove destination")
	err := os.RemoveAll(destination)
	span.Finish()
	if err != nil {
		return nil, errors.WithMessage(err, "remove existing destination")
	}
	span, _ = s.Tracer.FromCtx(ctx, "lock mutex")
	s.masterMutex.RLock()
	defer s.masterMutex.RUnlock()
	span.Finish()
	span, _ = s.Tracer.FromCtx(ctx, "copy to destination")
	err = s.Copier.CopyDir(ctx, s.MasterPath(), destination)
	span.Finish()
	if err != nil {
		return nil, errors.WithMessagef(err, "copy master from '%s'", s.MasterPath())
	}
	span, _ = s.Tracer.FromCtx(ctx, "open repo")
	r, err := git.PlainOpen(destination)
	span.Finish()
	if err != nil {
		return nil, errors.WithMessage(err, "open repo")
	}
	return r, nil
}

func (s *Service) Checkout(ctx context.Context, rootPath string, hash plumbing.Hash) error {
	span, ctx := s.Tracer.FromCtx(ctx, "git.Checkout")
	defer span.Finish()
	err := execCommand(ctx, rootPath, "git", "checkout", hash.String())
	if err != nil {
		return errors.WithMessage(err, "checkout hash")
	}
	return nil
}

// LocateRelease traverses the git log to find a release commit with id
// artifactID.
//
// It expects the commit to have a commit messages as the one returned by
// ReleaseCommitMessage.
func (s *Service) LocateRelease(ctx context.Context, r *git.Repository, artifactID string) (plumbing.Hash, error) {
	span, _ := s.Tracer.FromCtx(ctx, "git.LocateRelease")
	defer span.Finish()
	return locate(r, locateReleaseCondition(artifactID), ErrReleaseNotFound)
}

func locateReleaseCondition(artifactID string) conditionFunc {
	if artifactID == "" {
		return falseConditionFunc
	}
	return commitinfo.LocateRelease(func(c commitinfo.CommitInfo) bool {
		return strings.EqualFold(c.ArtifactID, artifactID)
	})
}

// LocateServiceRelease traverses the git log to find a release
// commit for a specified service and environment.
//
// It expects the commit to have a commit messages as the one returned by
// ReleaseCommitMessage.
func (s *Service) LocateServiceRelease(ctx context.Context, r *git.Repository, env, service string) (plumbing.Hash, error) {
	span, _ := s.Tracer.FromCtx(ctx, "git.LocateServiceRelease")
	defer span.Finish()
	return locate(r, locateServiceReleaseCondition(env, service), ErrReleaseNotFound)
}

func locateServiceReleaseCondition(env, service string) conditionFunc {
	if env == "" || service == "" {
		return falseConditionFunc
	}
	return commitinfo.LocateRelease(func(c commitinfo.CommitInfo) bool {
		return strings.EqualFold(c.Service, service) && strings.EqualFold(c.Environment, env)
	})
}

// LocateEnvRelease traverses the git log to find a release
// commit for a specified environment and artifactID.
//
// It expects the commit to have a commit messages as the one returned by
// ReleaseCommitMessage.
func (s *Service) LocateEnvRelease(ctx context.Context, r *git.Repository, env, artifactID string) (plumbing.Hash, error) {
	artifactID = strings.TrimSpace(artifactID)
	span, _ := s.Tracer.FromCtx(ctx, "git.LocateEnvRelease")
	defer span.Finish()
	return locate(r, locateEnvReleaseCondition(env, artifactID), ErrReleaseNotFound)
}

func locateEnvReleaseCondition(env, artifactID string) conditionFunc {
	if env == "" || artifactID == "" {
		return falseConditionFunc
	}
	return commitinfo.LocateRelease(func(c commitinfo.CommitInfo) bool {
		return strings.EqualFold(c.ArtifactID, artifactID) && strings.EqualFold(c.Environment, env)
	})
}

// LocateServiceReleaseRollbackSkip traverses the git log to find the nth
// release or rollback commit for a specified service and environment.
//
// It expects the commit to have a commit messages as the one returned by
// ReleaseCommitMessage or RollbackCommitMessage.
func (s *Service) LocateServiceReleaseRollbackSkip(ctx context.Context, r *git.Repository, env, service string, n uint) (plumbing.Hash, error) {
	span, _ := s.Tracer.FromCtx(ctx, "git.LocateServiceReleaseRollbackSkip")
	defer span.Finish()
	return locate(r, locateServiceReleaseRollbackSkipCondition(env, service, n), ErrReleaseNotFound)
}

func locateServiceReleaseRollbackSkipCondition(env, service string, n uint) conditionFunc {
	if env == "" || service == "" {
		return falseConditionFunc
	}
	return commitinfo.LocateRelease(func(c commitinfo.CommitInfo) bool {
		ok := strings.EqualFold(c.Environment, env) && strings.EqualFold(c.Service, service)
		if !ok {
			return false
		}
		if n == 0 {
			return true
		}
		n--
		return false
	})
}

type conditionFunc func(commitMsg string) bool

func locate(r *git.Repository, condition conditionFunc, notFoundErr error) (plumbing.Hash, error) {
	hashes, err := locateN(r, condition, notFoundErr, 1)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	// locateN will return an error when reaching the end of the git log.
	// So if there is no error we must have found at least one match.
	return hashes[0], nil
}

func locateN(r *git.Repository, condition conditionFunc, notFoundErr error, n int) ([]plumbing.Hash, error) {
	var hashes []plumbing.Hash
	ref, err := r.Head()
	if err != nil {
		return nil, errors.WithMessage(err, "retrieve HEAD branch")
	}
	cIter, err := r.Log(&git.LogOptions{
		From: ref.Hash(),
	})
	if err != nil {
		return nil, errors.WithMessage(err, "retrieve commit history")
	}
	for {
		commit, err := cIter.Next()
		if err != nil {
			if err == io.EOF {
				return hashes, notFoundErr
			}
			return nil, errors.WithMessage(err, "retrieve commit")
		}
		if condition(commit.Message) {
			hashes = append(hashes, commit.Hash)
		}
		if len(hashes) >= n {
			return hashes, nil
		}
	}
}

func (s *Service) Commit(ctx context.Context, rootPath, changesPath, msg string) error {
	span, ctx := s.Tracer.FromCtx(ctx, "git.Commit")
	defer span.Finish()

	span, _ = s.Tracer.FromCtx(ctx, "add changes")
	err := execCommand(ctx, rootPath, "git", "add", ".")
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "add changes")
	}

	span, _ = s.Tracer.FromCtx(ctx, "check for changes")
	err = checkStatus(ctx, rootPath)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "check for changes")
	}
	args := []string{
		"-c", fmt.Sprintf(`user.name="%s"`, s.Config.User),
		"-c", fmt.Sprintf(`user.email="%s"`, s.Config.Email),
		"commit",
	}

	if s.Config.SigningKey != "" {
		args = append(args, fmt.Sprintf("--gpg-sign=%s", s.Config.SigningKey))
	}
	args = append(args, fmt.Sprintf(`-m%s`, msg))

	span, _ = s.Tracer.FromCtx(ctx, "commit")
	err = execCommand(ctx, rootPath, "git", args...)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "commit")
	}

	span, _ = s.Tracer.FromCtx(ctx, "push")
	defer span.Finish()
	return gitPush(ctx, rootPath)
}

func (s *Service) SignedCommit(ctx context.Context, rootPath, changesPath, authorName, authorEmail, msg string) error {
	span, ctx := s.Tracer.FromCtx(ctx, "git.Commit")
	defer span.Finish()

	span, _ = s.Tracer.FromCtx(ctx, "add changes")
	err := execCommand(ctx, rootPath, "git", "add", ".")
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "add changes")
	}

	span, _ = s.Tracer.FromCtx(ctx, "check for changes")
	err = checkStatus(ctx, rootPath)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "check for changes")
	}
	fullCommitMsg := fmt.Sprintf("%s\nArtifact-created-by: %s <%s>", msg, authorName, authorEmail)
	span, _ = s.Tracer.FromCtx(ctx, "commit")
	err = execCommand(ctx, rootPath,
		"git",
		"commit",
		fmt.Sprintf(`-m%s`, fullCommitMsg),
	)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "commit")
	}

	span, _ = s.Tracer.FromCtx(ctx, "push")
	defer span.Finish()
	return gitPush(ctx, rootPath)
}

func execCommand(ctx context.Context, rootPath string, cmdName string, args ...string) error {
	logger := log.WithContext(ctx).WithFields("root", rootPath)
	logger.Infof("git/execCommand: running: %s %s", cmdName, strings.Join(args, " "))
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

	stdoutData, err := io.ReadAll(stdout)
	if err != nil {
		return errors.WithMessage(err, "read stdout data of command")
	}
	stderrData, err := io.ReadAll(stderr)
	if err != nil {
		return errors.WithMessage(err, "read stderr data of command")
	}

	err = cmd.Wait()
	logger.Infof("git/commit: exec command '%s %s': stdout: %s", cmdName, strings.Join(args, " "), stdoutData)
	logger.Infof("git/commit: exec command '%s %s': stderr: %s", cmdName, strings.Join(args, " "), stderrData)
	if err != nil {
		return errors.WithMessage(err, "execute command failed")
	}
	err = isKnownGitError(stderrData)
	if err != nil {
		return err
	}
	return nil
}

// knownGitErrors contains error messages that should be considered as errors by
// release-manager and because of this return an error.
var knownGitErrors = []string{
	"fatal: Could not read from remote repository.",
	"Connection closed by remote host",
	"ssh: Could not resolve hostname github.com",
}

// isKnownGitError returns an error if stderr is identified as a known Git
// error.
func isKnownGitError(stderrData []byte) error {
	if len(stderrData) == 0 {
		return nil
	}
	for _, e := range knownGitErrors {
		if bytes.Contains(stderrData, []byte(e)) {
			return ErrUnknownGit
		}
	}
	return nil
}

func checkStatus(ctx context.Context, rootPath string) error {
	cmdName := "git"
	args := []string{"status", "--porcelain"}
	logger := log.WithContext(ctx).WithFields("root", rootPath)
	logger.Infof("git/execCommand: running: %s %s", cmdName, strings.Join(args, " "))
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

	stdoutData, err := io.ReadAll(stdout)
	if err != nil {
		return errors.WithMessage(err, "read stdout data of command")
	}
	stderrData, err := io.ReadAll(stderr)
	if err != nil {
		return errors.WithMessage(err, "read stderr data of command")
	}

	err = cmd.Wait()
	logger.Infof("git/commit: exec command '%s %s': stdout: %s", cmdName, strings.Join(args, " "), stdoutData)
	logger.Infof("git/commit: exec command '%s %s': stderr: %s", cmdName, strings.Join(args, " "), stderrData)
	if err != nil {
		return errors.WithMessage(err, "execute command failed")
	}
	if len(stdoutData) == 0 {
		return ErrNothingToCommit
	}
	return nil
}

func gitPush(ctx context.Context, rootPath string) error {
	cmdName := "git"
	args := []string{"push", "origin", "master", "--porcelain"}
	logger := log.WithContext(ctx).WithFields("root", rootPath)
	logger.Infof("git/execCommand: running: %s %s", cmdName, strings.Join(args, " "))
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

	stdoutData, err := io.ReadAll(stdout)
	if err != nil {
		return errors.WithMessage(err, "read stdout data of command")
	}
	stderrData, err := io.ReadAll(stderr)
	if err != nil {
		return errors.WithMessage(err, "read stderr data of command")
	}

	err = cmd.Wait()
	logger.Infof("git/commit: exec command '%s %s': stdout: %s", cmdName, strings.Join(args, " "), stdoutData)
	logger.Infof("git/commit: exec command '%s %s': stderr: %s", cmdName, strings.Join(args, " "), stderrData)
	if err != nil {
		if isBranchBehindOrigin(stderrData) {
			return ErrBranchBehindOrigin
		}
		return errors.WithMessage(err, "execute command failed")
	}
	return nil
}

func falseConditionFunc(commitMsg string) bool { return false }

// branchBehindOriginIndicators contains partial Git messages that is used to
// detect branch behind origin errors on push commands.
var branchBehindOriginIndicators = []string{
	"rejected because the remote contains work that you do",
	"tip of your current branch is behind",
}

// isBranchBehindOrigin returns wether stderrData indicates that the push is
// rejected do to its local state is behind the origin.
func isBranchBehindOrigin(stderrData []byte) bool {
	if len(stderrData) == 0 {
		return false
	}

	// ignore casing of Git message to make it more rebust.
	stderrData = bytes.ToLower(stderrData)

	for _, e := range branchBehindOriginIndicators {
		if bytes.Contains(stderrData, []byte(e)) {
			return true
		}
	}
	return false
}
