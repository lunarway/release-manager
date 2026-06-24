package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// TestCommitAdvancesMirrorAfterPush verifies the self-staleness fix: after a
// normal (non-conflicting) push, Commit fast-forwards the local master mirror
// to the just-pushed commit, so a subsequent ShallowClone starts current rather
// than one commit behind origin. Without the advance, the mirror would lag by
// every release commit and the next push would be rejected as behind origin.
func TestCommitAdvancesMirrorAfterPush(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)

	ctx := context.Background()
	root := t.TempDir()

	originDir := filepath.Join(root, "origin")
	seedDir := filepath.Join(root, "seed")
	mirrorDir := filepath.Join(root, "mirror")
	workDir := filepath.Join(root, "work")

	runGit(t, root, "init", "--bare", "-b", "master", originDir)
	runGit(t, root, "clone", originDir, seedDir)
	configureIdentity(t, seedDir)
	writeFile(t, seedDir, "a.txt", "a")
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "add a")
	runGit(t, seedDir, "push", "origin", "master")

	// Full-history mirror, configured exactly as InitMasterRepo does so the
	// post-push advance can update the checked-out master branch in place.
	runGit(t, root, "clone", originDir, mirrorDir)
	runGit(t, mirrorDir, "config", "receive.denyCurrentBranch", "updateInstead")

	// Depth-1 shallow working clone from the current mirror.
	runGit(t, root, "clone", "--depth", "1", "--local", mirrorDir, workDir)
	runGit(t, workDir, "remote", "set-url", "origin", originDir)
	configureIdentity(t, workDir)

	writeFile(t, workDir, "c.txt", "c")

	s := &Service{
		Tracer:        tracing.NewNoop(),
		Config:        &GitConfig{User: "test", Email: "test@example.com"},
		ConfigRepoURL: originDir,
		masterPath:    mirrorDir,
	}

	err := s.Commit(ctx, workDir, ".", "[env/svc] release c")
	require.NoError(t, err)

	// The mirror must carry the release commit in both its ref and working tree,
	// so a later ShallowClone/copyMaster sees it without another origin fetch.
	assert.FileExists(t, filepath.Join(mirrorDir, "c.txt"),
		"push must fast-forward the local master mirror")
	assert.Equal(t, revParse(t, workDir, "HEAD"), revParse(t, mirrorDir, "master"),
		"mirror master must match the pushed commit")
}

// TestShallowCloneRecoveryUsesConfiguredIdentity reproduces the production
// failure where the rebase recovery path exits 128 ("unable to auto-detect
// email address"). Unlike the tests above, it builds the working tree via the
// real ShallowClone instead of manually configuring an identity, so the working
// tree carries only whatever identity ShallowClone sets. The rebase that
// recovers a push conflict writes a commit and therefore needs that identity;
// without it git aborts fatally and the release never recovers in place.
func TestShallowCloneRecoveryUsesConfiguredIdentity(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()

	// Force git to use only the configured identity, never an implicit
	// user@hostname one. This reproduces the container, where the hostname
	// resolves to ".(none)" and git refuses the auto-detected address with
	// "fatal: unable to auto-detect email address". Without this, a developer
	// machine's resolvable hostname would let the rebase commit succeed and mask
	// the bug.
	globalConfig := filepath.Join(root, "gitconfig")
	require.NoError(t, os.WriteFile(globalConfig, []byte("[user]\n\tuseConfigOnly = true\n"), 0o644))
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)

	originDir := filepath.Join(root, "origin")
	seedDir := filepath.Join(root, "seed")
	mirrorDir := filepath.Join(root, "mirror")
	workDir := filepath.Join(root, "work")

	runGit(t, root, "init", "--bare", "-b", "master", originDir)
	runGit(t, root, "clone", originDir, seedDir)
	configureIdentity(t, seedDir)
	writeFile(t, seedDir, "a.txt", "a")
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "add a")
	runGit(t, seedDir, "push", "origin", "master")

	runGit(t, root, "clone", originDir, mirrorDir)

	s := &Service{
		Tracer:        tracing.NewNoop(),
		Config:        &GitConfig{User: "test", Email: "test@example.com"},
		ConfigRepoURL: originDir,
		masterPath:    mirrorDir,
	}

	// Build the working tree exactly as the release flow does, via ShallowClone,
	// rather than cloning and configuring an identity by hand.
	err := s.ShallowClone(ctx, workDir)
	require.NoError(t, err)

	// An upstream commit lands after the clones were taken, so the first push
	// is rejected and Commit must rebase to recover.
	writeFile(t, seedDir, "b.txt", "b")
	runGit(t, seedDir, "add", ".")
	runGit(t, seedDir, "commit", "-m", "add b")
	runGit(t, seedDir, "push", "origin", "master")

	writeFile(t, workDir, "c.txt", "c")

	err = s.Commit(ctx, workDir, ".", "[env/svc] release c")
	require.NoError(t, err, "rebase recovery must use the configured identity, not abort with exit 128")

	pullLatest(t, seedDir)
	assert.FileExists(t, filepath.Join(seedDir, "b.txt"))
	assert.FileExists(t, filepath.Join(seedDir, "c.txt"))
}

func revParse(t *testing.T, dir, ref string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", ref)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git rev-parse %s failed: %s", ref, out)
	return strings.TrimSpace(string(out))
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
