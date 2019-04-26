package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	git "gopkg.in/src-d/go-git.v4"
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
	if err != nil {
		return "", errors.WithMessagef(err, "get commit at hash '%s'", h.Hash())
	}
	s, err := c.Stats()
	if err != nil {
		return "", errors.WithMessagef(err, "get stats at hash '%s'", h.Hash())
	}
	var modifiedFiles []string
	for _, s := range s {
		modifiedFiles = append(modifiedFiles, s.Name)
	}
	branchName, ok := BranchName(modifiedFiles, artifactFileName, svc)
	if !ok {
		return "", errors.New("branch not detectable")
	}
	return branchName, nil
}
