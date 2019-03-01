package git

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/pkg/errors"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func Clone(repoURL, destination string) (*git.Repository, error) {
	err := os.RemoveAll(destination)
	if err != nil {
		return nil, err
	}

	r, err := git.PlainClone(destination, false, &git.CloneOptions{
		URL: repoURL,
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

func LocateRelease(repoURL, release string) (plumbing.Hash, error) {
	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: repoURL,
	})
	if err != nil {
		return plumbing.ZeroHash, errors.WithMessage(err, "clone repo")
	}
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
				return plumbing.ZeroHash, errors.New("release not found")
			}
			return plumbing.ZeroHash, errors.WithMessage(err, "retrieve commit")
		}
		if strings.Contains(commit.Message, release) {
			return commit.Hash, nil
		}
	}
}

func Commit(repo *git.Repository, changesPath, service, env, tag, authorName, authorEmail string) error {
	w, err := repo.Worktree()
	if err != nil {
		return errors.WithMessage(err, "get worktree")
	}
	err = w.AddGlob(changesPath)
	if err != nil {
		return errors.WithMessage(err, "add changes")
	}
	err = Checkout(repo, plumbing.NewHash("HEAD"))
	if err != nil {
		return errors.WithMessage(err, "checkout HEAD")
	}
	_, err = w.Commit(fmt.Sprintf("[%s/%s] release tag %s by %s", env, service, tag, authorName), &git.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
		Committer: &object.Signature{
			Name:  "HamAstrochimp",
			Email: "operations@lunarway.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return errors.WithMessage(err, "commit")
	}
	return nil
}
