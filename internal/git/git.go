package git

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"runtime"
	"time"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/format/config"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

var (
	ErrNothingToCommit  = errors.New("nothing to commit")
	ErrReleaseNotFound  = errors.New("release not found")
	ErrArtifactNotFound = errors.New("artifact not found")
)

func CloneDepth(ctx context.Context, repoURL, destination, sshPrivateKeyPath string, depth int) (*git.Repository, error) {
	return clone(ctx, repoURL, destination, sshPrivateKeyPath, depth)
}
func Clone(ctx context.Context, repoURL, destination, sshPrivateKeyPath string) (*git.Repository, error) {
	return clone(ctx, repoURL, destination, sshPrivateKeyPath, 0)
}

func clone(ctx context.Context, repoURL, destination, sshPrivateKeyPath string, depth int) (*git.Repository, error) {
	authSSH, err := ssh.NewPublicKeysFromFile("git", sshPrivateKeyPath, "")
	if err != nil {
		return nil, errors.WithMessage(err, "public keys from file")
	}
	err = os.RemoveAll(destination)
	if err != nil {
		return nil, errors.WithMessage(err, "remove existing destination")
	}

	r, err := git.PlainCloneContext(ctx, destination, false, &git.CloneOptions{
		URL:   repoURL,
		Auth:  authSSH,
		Depth: depth,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "clone repo")
	}
	return r, nil
}

func Checkout(r *git.Repository, hash plumbing.Hash) error {
	workTree, err := r.Worktree()
	if err != nil {
		return errors.WithMessage(err, "get worktree")
	}
	err = workTree.Checkout(&git.CheckoutOptions{
		Hash: hash,
	})
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
func LocateRelease(r *git.Repository, artifactID string) (plumbing.Hash, error) {
	return locate(r, locateReleaseCondition(artifactID), ErrReleaseNotFound)
}

func locateReleaseCondition(artifactID string) conditionFunc {
	r := regexp.MustCompile(fmt.Sprintf(`(?i)release %s$`, regexp.QuoteMeta(artifactID)))
	return func(commitMsg string) bool {
		if artifactID == "" {
			return false
		}
		return r.MatchString(commitMsg)
	}
}

// LocateServiceRelease traverses the git log to find a release
// commit for a specified service and environment.
//
// It expects the commit to have a commit messages as the one returned by
// ReleaseCommitMessage.
func LocateServiceRelease(r *git.Repository, env, service string) (plumbing.Hash, error) {
	return locate(r, locateServiceReleaseCondition(env, service), ErrReleaseNotFound)
}

func locateServiceReleaseCondition(env, service string) conditionFunc {
	r := regexp.MustCompile(fmt.Sprintf(`(?i)\[%s/%s] release`, regexp.QuoteMeta(env), regexp.QuoteMeta(service)))
	return func(commitMsg string) bool {
		if env == "" {
			return false
		}
		if service == "" {
			return false
		}
		return r.MatchString(commitMsg)
	}
}

// LocateEnvRelease traverses the git log to find a release
// commit for a specified environment and artifactID.
//
// It expects the commit to have a commit messages as the one returned by
// ReleaseCommitMessage.
func LocateEnvRelease(r *git.Repository, env, artifactID string) (plumbing.Hash, error) {
	return locate(r, locateEnvReleaseCondition(env, artifactID), ErrReleaseNotFound)
}

func locateEnvReleaseCondition(env, artifactId string) conditionFunc {
	r := regexp.MustCompile(fmt.Sprintf(`(?i)\[%s/.*] release %s$`, regexp.QuoteMeta(env), regexp.QuoteMeta(artifactId)))
	return func(commitMsg string) bool {
		if env == "" {
			return false
		}
		if artifactId == "" {
			return false
		}
		return r.MatchString(commitMsg)
	}
}

// LocateServiceReleaseRollbackSkip traverses the git log to find a release or
// rollback commit for a specified service and environment.
//
// It expects the commit to have a commit messages as the one returned by
// ReleaseCommitMessage or RollbackCommitMessage.
func LocateServiceReleaseRollbackSkip(r *git.Repository, env, service string, n uint) (plumbing.Hash, error) {
	return locate(r, locateServiceReleaseRollbackSkipCondition(env, service, n), ErrReleaseNotFound)
}

func locateServiceReleaseRollbackSkipCondition(env, service string, n uint) conditionFunc {
	return func(commitMsg string) bool {
		releaseOK := locateServiceReleaseCondition(env, service)(commitMsg)
		rollbackOK := locateServiceRollbackCondition(env, service)(commitMsg)
		ok := releaseOK || rollbackOK
		if !ok {
			return false
		}
		if n == 0 {
			return true
		}
		n--
		return false
	}
}

func locateServiceRollbackCondition(env, service string) conditionFunc {
	r := regexp.MustCompile(fmt.Sprintf(`(?i)\[%s/%s] rollback `, regexp.QuoteMeta(env), regexp.QuoteMeta(service)))
	return func(commitMsg string) bool {
		if env == "" {
			return false
		}
		if service == "" {
			return false
		}
		return r.MatchString(commitMsg)
	}
}

// LocateArtifact traverses the git log to find an artifact commit with id
// artifactID.
//
// It expects the commit to have a commit messages as the one returned by
// ArtifactCommitMessage.
func LocateArtifact(r *git.Repository, artifactID string) (plumbing.Hash, error) {
	return locate(r, locateArtifactCondition(artifactID), ErrArtifactNotFound)
}

func locateArtifactCondition(artifactID string) conditionFunc {
	artifactRegex := regexp.MustCompile(fmt.Sprintf(`(?i)artifact %s `, regexp.QuoteMeta(artifactID)))
	return func(commitMsg string) bool {
		if artifactID == "" {
			return false
		}
		return artifactRegex.MatchString(commitMsg)
	}
}

type conditionFunc func(commitMsg string) bool

func locate(r *git.Repository, condition conditionFunc, notFoundErr error) (plumbing.Hash, error) {
	ref, err := r.Head()
	if err != nil {
		return plumbing.ZeroHash, errors.WithMessage(err, "retrieve HEAD branch")
	}
	cIter, err := r.Log(&git.LogOptions{
		From: ref.Hash(),
	})
	if err != nil {
		return plumbing.ZeroHash, errors.WithMessage(err, "retrieve commit history")
	}
	for {
		commit, err := cIter.Next()
		if err != nil {
			if err == io.EOF {
				return plumbing.ZeroHash, notFoundErr
			}
			return plumbing.ZeroHash, errors.WithMessage(err, "retrieve commit")
		}
		if condition(commit.Message) {
			return commit.Hash, nil
		}
	}
}

func Commit(ctx context.Context, repo *git.Repository, changesPath, authorName, authorEmail, committerName, committerEmail, msg, sshPrivateKeyPath string) error {
	w, err := repo.Worktree()
	if err != nil {
		return errors.WithMessage(err, "get worktree")
	}
	err = w.AddGlob(changesPath)
	if err != nil {
		return errors.WithMessage(err, "add changes")
	}

	status, err := w.Status()
	if err != nil {
		return errors.WithMessage(err, "status")
	}
	log.Infof("internal/git: Commit status:\n%s", status)
	// if commit is empty
	if status.IsClean() {
		log.Debugf("internal/git: Commit: message '%s': nothing to commit", msg)
		return ErrNothingToCommit
	}

	_, err = w.Commit(msg, &git.CommitOptions{
		All: true,
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
		Committer: &object.Signature{
			Name:  committerName,
			Email: committerEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return errors.WithMessage(err, "commit")
	}

	authSSH, err := ssh.NewPublicKeysFromFile("git", sshPrivateKeyPath, "")
	if err != nil {
		return errors.WithMessage(err, "public keys from file")
	}

	// TODO: this could be made optional if needed
	err = repo.PushContext(ctx, &git.PushOptions{Auth: authSSH})
	if err != nil {
		return errors.WithMessage(err, "push")
	}
	return nil
}

// userHomeDir returns the home directory of the current user.
//
// It handles windows, linux and darwin operating systems by inspecting
// runtime.GOOS.
func userHomeDir() string {
	switch runtime.GOOS {
	case "windows":
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	case "linux":
		home := os.Getenv("XDG_CONFIG_HOME")
		if home != "" {
			return home
		}
		fallthrough
	default:
		return os.Getenv("HOME")
	}
}

// CommitterDetails returns name and email read for a Git configuration file.
//
// Configuration files are read first in the local git repository (if available)
// and then read the global Git configuration.
func CommitterDetails() (string, string, error) {
	var paths []string
	pwd, err := os.Getwd()
	if err == nil {
		paths = append(paths, path.Join(pwd, ".git", "config"))
	}
	paths = append(paths, path.Join(userHomeDir(), ".gitconfig"))
	return credentials(paths...)
}

// credentials will try to read user name and email from provided paths.
func credentials(paths ...string) (string, string, error) {
	for _, path := range paths {
		c, err := parseConfig(path)
		if err != nil {
			continue
		}
		committerName := c.Section("user").Option("name")
		committerEmail := c.Section("user").Option("email")
		if committerEmail == "" {
			continue
		}
		if committerName == "" {
			continue
		}
		return committerName, committerEmail, nil
	}
	return "", "", errors.Errorf("failed to read Git credentials from paths: %v", paths)
}

// parseConfig returns the Git configuration parsed from provided path.
func parseConfig(path string) (config.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return config.Config{}, err
	}
	decoder := config.NewDecoder(file)
	var c config.Config
	err = decoder.Decode(&c)
	if err != nil {
		return config.Config{}, err
	}
	return c, nil
}
