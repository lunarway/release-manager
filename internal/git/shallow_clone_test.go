package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShallowCloneIsFullHistoryHardlink locks in optimisation A: ShallowClone
// drops --depth 1 and keeps --local, so the resulting working clone carries the
// full mirror history and git hardlinks the object store instead of copying it.
// The full-history assertion is the behavioural discriminator against the old
// depth-1 clone; the shared-inode assertion proves the clone stayed a cheap
// hardlink rather than a full object copy.
func TestShallowCloneIsFullHistoryHardlink(t *testing.T) {
	// Isolate from the developer's global/system git config so the production
	// exec paths behave deterministically.
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)

	ctx := context.Background()
	root := t.TempDir()

	originDir := filepath.Join(root, "origin")
	seedDir := filepath.Join(root, "seed")
	mirrorDir := filepath.Join(root, "mirror")
	workDir := filepath.Join(root, "work")

	// Bare origin (stands in for the config repo) with two commits so a depth-1
	// clone would visibly truncate history.
	runGit(t, root, "init", "--bare", "-b", "master", originDir)
	runGit(t, root, "clone", originDir, seedDir)
	configureIdentity(t, seedDir)
	writeFile(t, seedDir, "a.txt", "a")
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "add a")
	writeFile(t, seedDir, "b.txt", "b")
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "add b")
	runGit(t, seedDir, "push", "origin", "master")

	// Full-history local mirror (Service.masterPath) cloned from origin.
	runGit(t, root, "clone", originDir, mirrorDir)

	s := &Service{
		Tracer:        tracing.NewNoop(),
		Config:        &GitConfig{User: "test", Email: "test@example.com"},
		ConfigRepoURL: originDir,
		masterPath:    mirrorDir,
	}

	err := s.ShallowClone(ctx, workDir)
	require.NoError(t, err)

	// Full history: both commits are present, unlike a depth-1 clone.
	assert.Equal(t, 2, commitCount(t, workDir),
		"clone must retain full history, not a single depth-1 commit")

	// The origin URL is repointed at the real config repo for later pushes.
	assert.Equal(t, originDir, remoteURL(t, workDir, "origin"))

	// Hardlinked object store: at least one object file in the clone shares an
	// inode with the mirror, proving --local hardlinked rather than copied.
	assert.True(t, sharesObjectInode(t, mirrorDir, workDir),
		"clone object store must be hardlinked to the mirror, not a full copy")
}

func commitCount(t *testing.T, dir string) int {
	t.Helper()
	out := gitOutput(t, dir, "rev-list", "--count", "HEAD")
	count, err := strconv.Atoi(out)
	require.NoErrorf(t, err, "unexpected rev-list output: %q", out)
	return count
}

func remoteURL(t *testing.T, dir, remote string) string {
	t.Helper()
	return gitOutput(t, dir, "remote", "get-url", remote)
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v failed: %s", args, out)
	return strings.TrimSpace(string(out))
}

// sharesObjectInode reports whether any object file under dst/.git/objects is
// the same inode as a file under src/.git/objects, which is what a --local
// hardlink clone produces.
func sharesObjectInode(t *testing.T, src, dst string) bool {
	t.Helper()
	srcInodes := objectInodes(t, src)
	for ino := range objectInodes(t, dst) {
		if srcInodes[ino] {
			return true
		}
	}
	return false
}

func objectInodes(t *testing.T, repo string) map[uint64]bool {
	t.Helper()
	inodes := make(map[uint64]bool)
	objects := filepath.Join(repo, ".git", "objects")
	err := filepath.Walk(objects, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if st, ok := info.Sys().(*syscall.Stat_t); ok {
			inodes[st.Ino] = true
		}
		return nil
	})
	require.NoError(t, err)
	return inodes
}
