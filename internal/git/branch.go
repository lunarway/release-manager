package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// BranchName returns the branch name and a bool indicating one is found from a
// list of modified file paths.
//
// It only handles files that originates from a build operation, ie. non-build
// commits cannot be extracted.
func BranchName(modifiedFiles []string, artifactFileName, svc string) (string, bool) {
	var branch string
	for _, f := range modifiedFiles {
		if !strings.Contains(f, artifactFileName) {
			continue
		}
		branch = strings.TrimPrefix(f, fmt.Sprintf("artifacts/%s/", svc))
		break
	}
	if len(branch) == 0 {
		return "", false
	}
	return strings.TrimSuffix(branch, fmt.Sprintf("/%s", artifactFileName)), true
}

// BranchFromHead reutrns the branch name from the current HEAD commit.
//
// It only handles files that originates from a build operation, ie. non-build
// commits cannot be extracted.
func BranchFromHead(ctx context.Context, repo *git.Repository, artifactFileName, svc string) (string, error) {
	h, err := repo.Head()
	if err != nil {
		return "", errors.WithMessage(err, "get worktree")
	}
	c, err := repo.CommitObject(h.Hash())
	files, err := c.Files()
	var modifiedFiles []string

	files.ForEach(func(f *object.File) error {
		modifiedFiles = append(modifiedFiles, f.Name)
		return nil
	})
	branchName, ok := BranchName(modifiedFiles, artifactFileName, svc)
	if !ok {
		return "", errors.New("branch not detectable")
	}
	return branchName, nil
}
