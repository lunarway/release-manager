package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestMain(m *testing.M) {
	log.Init(&log.Configuration{
		Level:       log.Level{Level: zapcore.DebugLevel},
		Development: true,
	})
	os.Exit(m.Run())
}

// TestCommitRebasesOntoSyncedMirrorOnConflict reproduces the release-flow
// conflict path: a depth-1 shallow working clone whose local master mirror is
// stale relative to origin. On push rejection Commit must refresh the mirror,
// rebase the working tree onto it, and re-push in place without falling back to
// a full re-clone. The mirror ending up with the upstream commit is what proves
// the in-place sync+rebase recovery ran.
func TestCommitRebasesOntoSyncedMirrorOnConflict(t *testing.T) {
	// Isolate from the developer's global/system git config (e.g. commit.gpgsign)
	// so the production exec paths behave deterministically.
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)

	ctx := context.Background()
	root := t.TempDir()

	originDir := filepath.Join(root, "origin")
	seedDir := filepath.Join(root, "seed")
	mirrorDir := filepath.Join(root, "mirror")
	workDir := filepath.Join(root, "work")

	// Bare origin (stands in for the config repo) with an initial commit.
	runGit(t, root, "init", "--bare", "-b", "master", originDir)
	runGit(t, root, "clone", originDir, seedDir)
	configureIdentity(t, seedDir)
	writeFile(t, seedDir, "a.txt", "a")
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "add a")
	runGit(t, seedDir, "push", "origin", "master")

	// Full-history local mirror (Service.masterPath) cloned from origin.
	runGit(t, root, "clone", originDir, mirrorDir)

	// Depth-1 shallow working clone from the mirror, then point origin at the
	// real config repo, exactly as ShallowClone does.
	runGit(t, root, "clone", "--depth", "1", "--local", mirrorDir, workDir)
	runGit(t, workDir, "remote", "set-url", "origin", originDir)
	configureIdentity(t, workDir)

	// An upstream commit lands on origin after the mirror and working clone were
	// taken, leaving both stale.
	writeFile(t, seedDir, "b.txt", "b")
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "add b")
	runGit(t, seedDir, "push", "origin", "master")

	// The release change in the working tree.
	writeFile(t, workDir, "c.txt", "c")

	s := &Service{
		Tracer:        tracing.NewNoop(),
		Config:        &GitConfig{User: "test", Email: "test@example.com"},
		ConfigRepoURL: originDir,
		masterPath:    mirrorDir,
	}

	err := s.Commit(ctx, workDir, ".", "[env/svc] release c")
	require.NoError(t, err, "commit should recover from the push conflict in place")

	// Origin now carries the upstream commit and the release commit: the working
	// tree was rebased and re-pushed, not abandoned.
	assert.FileExists(t, filepath.Join(seedDir, "a.txt"))
	pullLatest(t, seedDir)
	assert.FileExists(t, filepath.Join(seedDir, "b.txt"))
	assert.FileExists(t, filepath.Join(seedDir, "c.txt"))

	// The mirror was synced as part of the recovery. The old rebase-onto-origin
	// path never touched the mirror, so this is the behavioural discriminator.
	assert.FileExists(t, filepath.Join(mirrorDir, "b.txt"),
		"recovery must sync the local master mirror")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v failed: %s", args, out)
}

func configureIdentity(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "config", "user.name", "test")
	runGit(t, dir, "config", "user.email", "test@example.com")
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func pullLatest(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "pull", "origin", "master")
}
