package git

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

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
	return locate(r, func(commitMsg string) bool {
		return strings.Contains(commitMsg, artifactID)
	}, ErrReleaseNotFound)
}

// LocateArtifact traverses the git log to find an artifact commit with id
// artifactID.
//
// It expects the commit to have a commit messages as the one returned by
// ArtifactCommitMessage.
func LocateArtifact(r *git.Repository, artifactID string) (plumbing.Hash, error) {
	return locate(r, func(commitMsg string) bool {
		return strings.Contains(commitMsg, fmt.Sprintf("artifact %s", artifactID))
	}, ErrArtifactNotFound)
}

func locate(r *git.Repository, condition func(commitMsg string) bool, notFoundErr error) (plumbing.Hash, error) {
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

	// if commit is empty
	if status.IsClean() {
		return ErrNothingToCommit
	}

	_, err = w.Commit(msg, &git.CommitOptions{
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

// GlobalConfig returns the global Git configuration read from the user home
// directory.
func GlobalConfig() (config.Config, error) {
	file, err := os.Open(path.Join(userHomeDir(), ".gitconfig"))
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

func CommitterDetails() (string, string, error) {
	c, err := GlobalConfig()
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
