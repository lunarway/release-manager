package flow_test

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	internalgit "github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/stretchr/testify/assert"
)

// TestService_DescribeRelease_basicFlow tests the basic git flow of the
// DescribeRelease method. It does not test the actual implementation as that
// depends on the filesystem so this is a best effort on locking and verifying
// the behaviour.
//
// The test was introduced when debugging an issue where some releases were not
// reported in the results due to how the flow was handling checkouts.
func TestService_DescribeRelease_basicFlow(t *testing.T) {
	// setup temporary repository
	tmpDir, err := ioutil.TempDir("", "release-manager-test")
	if err != nil {
		t.Fatalf("failed to get temp dir: %v", err)
	}
	defer func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			t.Fatalf("failed to remove temp dir with repository: %v", err)
		}
	}()

	repo, err := git.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	// setup commits in the repository for testing
	commits := []struct {
		message string
		hash    plumbing.Hash
	}{
		{
			message: "[prod/user] release master-1 by obr@lunar.app",
		},
		{
			message: "[prod/user] release master-2 by shf@lunar.app",
		},
		{
			message: "[prod/user] release master-3 by rol@lunar.app",
		},
		{
			message: "[prod/user] release master-4 by tss@lunar.app",
		},
	}
	for i := range commits {
		hash, err := wt.Commit(commits[i].message, &git.CommitOptions{})
		if err != nil {
			t.Fatalf("failed to commit to worktree: %v", err)
		}
		commits[i].hash = hash
	}

	// print the commit log for human inspection on failures
	t.Log("Commit log")
	iter, err := repo.Log(&git.LogOptions{})
	if err != nil {
		t.Fatalf("failed to get repo log: %v", err)
	}
	iter.ForEach(func(c *object.Commit) error {
		t.Logf("- %s %s", c.Hash.String(), c.Message)
		return nil
	})

	s := internalgit.Service{
		Tracer: tracing.NewNoop(),
	}

	// this emulates the core loop of the DescribeRelease flow
	t.Logf("Finding releases")
	for i := 0; i < len(commits); i++ {
		hash, err := s.LocateServiceReleaseRollbackSkip(context.Background(), repo, "prod", "user", uint(i))
		if err != nil {
			t.Fatalf("failed to locate release: %v", err)
		}

		t.Logf("- %s n=%d", hash.String(), i)
		assert.Equal(t, commits[len(commits)-(i+1)].hash.String(), hash.String(), "found hash not as expected for i=%d", i)

		err = wt.Checkout(&git.CheckoutOptions{
			Hash: hash,
		})
		if err != nil {
			t.Fatalf("failed to checkout hash: %v", err)
		}

		// checkout master again to reset HEAD
		err = wt.Checkout(&git.CheckoutOptions{
			Branch: plumbing.Master,
		})
		if err != nil {
			t.Fatalf("failed to checkout master: %v", err)
		}
	}
}
