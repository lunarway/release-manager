package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocateReleaseCondition(t *testing.T) {
	tt := []struct {
		name       string
		artifactID string
		message    string
		output     bool
	}{
		{
			name:       "empty artifact ID",
			artifactID: "",
			message:    "[env/service-name] release master-1234567890-1234567890",
			output:     false,
		},
		{
			name:       "regexp like artifact id",
			artifactID: `(\`,
			message:    "[env/service-name] release master-1234567890-1234567890",
			output:     false,
		},
		{
			name:       "partial artifact id",
			artifactID: "master-1234",
			message:    "[env/service-name] release master-1234567890-1234567890",
			output:     false,
		},
		{
			name:       "partial artifact id with complete application hash",
			artifactID: "master-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890",
			output:     false,
		},
		{
			name:       "exact artifact id",
			artifactID: "master-1234567890-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890",
			output:     true,
		},
		{
			name:       "wrong cased artifact id",
			artifactID: "MASTER-1234567890-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890",
			output:     true,
		},
		{
			name:       "empty artifact ID and author email",
			artifactID: "",
			message:    "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			output:     false,
		},
		{
			name:       "regexp like artifact id and author email",
			artifactID: `(\`,
			message:    "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			output:     false,
		},
		{
			name:       "partial artifact id and author email",
			artifactID: "master-1234",
			message:    "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			output:     false,
		},
		{
			name:       "partial artifact id with complete application hash and author email",
			artifactID: "master-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			output:     false,
		},
		{
			name:       "exact artifact id and author email",
			artifactID: "master-1234567890-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			output:     true,
		},
		{
			name:       "wrong cased artifact id and author email",
			artifactID: "MASTER-1234567890-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			output:     true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			output := locateReleaseCondition(tc.artifactID)(tc.message)
			assert.Equal(t, tc.output, output, "output not as expected")
		})
	}
}

func TestLocateServiceReleaseCondition(t *testing.T) {
	tt := []struct {
		name    string
		env     string
		service string
		message string
		output  bool
	}{
		{
			name:    "empty env",
			env:     "",
			service: "service-name",
			message: "[env/service-name] release master-1234567890-1234567890",
			output:  false,
		},
		{
			name:    "empty service",
			env:     "env",
			service: "",
			message: "[env/service-name] release master-1234567890-1234567890",
			output:  false,
		},
		{
			name:    "regexp like env",
			env:     `(\`,
			service: "",
			message: "[env/service-name] release master-1234567890-1234567890",
			output:  false,
		},
		{
			name:    "regexp like service",
			env:     "",
			service: `(\`,
			message: "[env/service-name] release master-1234567890-1234567890",
			output:  false,
		},
		{
			name:    "partial env",
			env:     "nv",
			service: "service-name",
			message: "[env/service-name] release master-1234567890-1234567890",
			output:  false,
		},
		{
			name:    "partial service",
			env:     "env",
			service: "service",
			message: "[env/service-name] release master-1234567890-1234567890",
			output:  false,
		},
		{
			name:    "exact env and service",
			env:     "env",
			service: "service-name",
			message: "[env/service-name] release master-1234567890-1234567890",
			output:  true,
		},
		{
			name:    "wrong cased env",
			env:     "ENV",
			service: "service-name",
			message: "[env/service-name] release master-1234567890-1234567890",
			output:  true,
		},
		{
			name:    "wrong cased service",
			env:     "env",
			service: "SERVICE-NAME",
			message: "[env/service-name] release master-1234567890-1234567890",
			output:  true,
		},
		{
			name:    "exact env and service and author email",
			env:     "env",
			service: "service-name",
			message: "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			output:  true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			output := locateServiceReleaseCondition(tc.env, tc.service)(tc.message)
			assert.Equal(t, tc.output, output, "output not as expected")
		})
	}
}

func TestLocateServiceReleaseRollbackSkipCondition(t *testing.T) {
	type result struct {
		commitMessage string
		located       bool
	}
	tt := []struct {
		name    string
		env     string
		service string
		skip    uint
		cases   []result
	}{
		{
			name:    "empty env",
			env:     "",
			service: "service",
			skip:    0,
			cases: []result{
				{"[env/service-name] release master-1234567890-1234567890", false},
			},
		},
		{
			name:    "empty service",
			env:     "env",
			service: "",
			skip:    0,
			cases: []result{
				{"[env/service-name] release master-1234567890-1234567890", false},
			},
		},
		{
			name:    "exact release commit on first case and 0 skip",
			env:     "env",
			service: "service-name",
			skip:    0,
			cases: []result{
				{"[env/service-name] release master-1234567890-1234567890", true},
				{"[env/service-name] release master-0123456789-0123456789", false},
			},
		},
		{
			name:    "exact release commit on second case and 1 skip",
			env:     "env",
			service: "service-name",
			skip:    1,
			cases: []result{
				{"[env/service-name] release master-1234567890-1234567890", false},
				{"[env/service-name] release master-0123456789-0123456789", true},
			},
		},
		{
			name:    "wrong case release commit on second case and 1 skip",
			env:     "env",
			service: "SERVICE-NAME",
			skip:    1,
			cases: []result{
				{"[env/service-name] release master-1234567890-1234567890", false},
				{"[env/service-name] release master-0123456789-0123456789", true},
			},
		},
		{
			name:    "exact rollback commit on second case and 1 skip",
			env:     "env",
			service: "service-name",
			skip:    1,
			cases: []result{
				{"[env/service-name] release master-1234567890-1234567890", false},
				{"[env/service-name] rollback master-1234567890-1234567890 to master-0123456789-0123456789", true},
			},
		},
		{
			name:    "wrong case service rollback commit on second case and 1 skip",
			env:     "env",
			service: "SERVICE-NAME",
			skip:    1,
			cases: []result{
				{"[env/service-name] release master-1234567890-1234567890", false},
				{"[env/service-name] rollback master-1234567890-1234567890 to master-0123456789-0123456789", true},
			},
		},
		{
			name:    "empty env and author email",
			env:     "",
			service: "service",
			skip:    0,
			cases: []result{
				{"[env/service-name] release master-1234567890-1234567890 by test@lunar.app", false},
			},
		},
		{
			name:    "empty service and author email",
			env:     "env",
			service: "",
			skip:    0,
			cases: []result{
				{"[env/service-name] release master-1234567890-1234567890 by test@lunar.app", false},
			},
		},
		{
			name:    "exact release commit on first case and 0 skip and author email",
			env:     "env",
			service: "service-name",
			skip:    0,
			cases: []result{
				{"[env/service-name] release master-1234567890-1234567890 by test@lunar.app", true},
				{"[env/service-name] release master-0123456789-0123456789 by test@lunar.app", false},
			},
		},
		{
			name:    "exact release commit on second case and 1 skip and author email",
			env:     "env",
			service: "service-name",
			skip:    1,
			cases: []result{
				{"[env/service-name] release master-1234567890-1234567890 by test@lunar.app", false},
				{"[env/service-name] release master-0123456789-0123456789 by test@lunar.app", true},
			},
		},
		{
			name:    "wrong case release commit on second case and 1 skip and author email",
			env:     "env",
			service: "SERVICE-NAME",
			skip:    1,
			cases: []result{
				{"[env/service-name] release master-1234567890-1234567890 by test@lunar.app", false},
				{"[env/service-name] release master-0123456789-0123456789 by test@lunar.app", true},
			},
		},
		{
			name:    "exact rollback commit on second case and 1 skip and author email",
			env:     "env",
			service: "service-name",
			skip:    1,
			cases: []result{
				{"[env/service-name] release master-1234567890-1234567890 by test@lunar.app", false},
				{"[env/service-name] rollback master-1234567890-1234567890 to master-0123456789-0123456789 by test@lunar.app", true},
			},
		},
		{
			name:    "wrong case service rollback commit on second case and 1 skip and author email",
			env:     "env",
			service: "SERVICE-NAME",
			skip:    1,
			cases: []result{
				{"[env/service-name] release master-1234567890-1234567890 by test@lunar.app", false},
				{"[env/service-name] rollback master-1234567890-1234567890 to master-0123456789-0123456789 by test@lunar.app", true},
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			f := locateServiceReleaseRollbackSkipCondition(tc.env, tc.service, tc.skip)
			for _, c := range tc.cases {
				output := f(c.commitMessage)
				if assert.Equalf(t, c.located, output, "output not as expected for message '%s'", c.commitMessage) {
					// break on first successful condition
					// this mimiks the logic of locate()
					if output {
						break
					}
				}
			}
		})
	}
}

func TestLocateEnvReleaseCondition(t *testing.T) {
	tt := []struct {
		name       string
		env        string
		artifactID string
		message    string
		output     bool
	}{
		{
			name:       "empty env",
			env:        "",
			artifactID: "master-1234567890-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890",
			output:     false,
		},
		{
			name:       "empty artifactID",
			env:        "env",
			artifactID: "",
			message:    "[env/service-name] release master-1234567890-1234567890",
			output:     false,
		},
		{
			name:       "regexp like env",
			env:        `(\`,
			artifactID: "master-1234567890-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890",
			output:     false,
		},
		{
			name:       "regexp like artifactId",
			env:        "",
			artifactID: `(\`,
			message:    "[env/service-name] release master-1234567890-1234567890",
			output:     false,
		},
		{
			name:       "partial env",
			env:        "nv",
			artifactID: "master-1234567890-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890",
			output:     false,
		},
		{
			name:       "partial artifactId",
			env:        "env",
			artifactID: "master-12345",
			message:    "[env/service-name] release master-1234567890-1234567890",
			output:     false,
		},
		{
			name:       "exact env and service",
			env:        "env",
			artifactID: "master-1234567890-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890",
			output:     true,
		},
		{
			name:       "wrong cased env",
			env:        "ENV",
			artifactID: "master-1234567890-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890",
			output:     true,
		},
		{
			name:       "wrong cased service",
			env:        "env",
			artifactID: "MASTER-1234567890-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890",
			output:     true,
		},
		{
			name:       "trailing newline",
			env:        "env",
			artifactID: "MASTER-1234567890-1234567890",
			message: `[env/service-name] release master-1234567890-1234567890
`,
			output: true,
		},
		{
			name:       "empty env and author email",
			env:        "",
			artifactID: "master-1234567890-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			output:     false,
		},
		{
			name:       "empty artifactID and author email",
			env:        "env",
			artifactID: "",
			message:    "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			output:     false,
		},
		{
			name:       "regexp like env and author email",
			env:        `(\`,
			artifactID: "master-1234567890-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			output:     false,
		},
		{
			name:       "regexp like artifactId and author email",
			env:        "",
			artifactID: `(\`,
			message:    "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			output:     false,
		},
		{
			name:       "partial env and author email",
			env:        "nv",
			artifactID: "master-1234567890-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			output:     false,
		},
		{
			name:       "partial artifactId and author email",
			env:        "env",
			artifactID: "master-12345",
			message:    "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			output:     false,
		},
		{
			name:       "exact env and service and author email",
			env:        "env",
			artifactID: "master-1234567890-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			output:     true,
		},
		{
			name:       "wrong cased env and author email",
			env:        "ENV",
			artifactID: "master-1234567890-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			output:     true,
		},
		{
			name:       "wrong cased service and author email",
			env:        "env",
			artifactID: "MASTER-1234567890-1234567890",
			message:    "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			output:     true,
		},
		{
			name:       "trailing newline and author email",
			env:        "env",
			artifactID: "MASTER-1234567890-1234567890",
			message: `[env/service-name] release master-1234567890-1234567890 by test@lunar.app
`,
			output: true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			output := locateEnvReleaseCondition(tc.env, tc.artifactID)(tc.message)
			assert.Equal(t, tc.output, output, "output not as expected")
		})
	}
}

func TestIsKnownGitError(t *testing.T) {
	tt := []struct {
		name   string
		stderr string
		err    error
	}{
		{
			name:   "no data",
			stderr: "",
			err:    nil,
		},
		{
			name:   "something unknown",
			stderr: "something we have never seen before",
			err:    nil,
		},
		{
			name:   "connection closed by remote",
			stderr: "ssh_exchange_idedntification: Connection closed by remote host",
			err:    ErrUnknownGit,
		},
		{
			name:   "hostname not resolved",
			stderr: "ssh: Could not resolve hostname github.com",
			err:    ErrUnknownGit,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := isKnownGitError([]byte(tc.stderr))
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error(), "error not as expected")
			} else {
				assert.NoError(t, err, "unexpected error")
			}
		})
	}
}

func TestIsBranchBehindOrigin(t *testing.T) {
	tt := []struct {
		name     string
		stderr   string
		isBehind bool
	}{
		{
			name:     "empty string",
			stderr:   "",
			isBehind: false,
		},
		{
			name:     "unknown git error",
			stderr:   `fatal: Could not read from remote repository.`,
			isBehind: false,
		},
		{
			name: "tip behind",
			stderr: `error: failed to push some refs to 'git@github.com:lunarway/k8s-cluster-config.git'
hint: Updates were rejected because the tip of your current branch is behind
hint: its remote counterpart. Integrate the remote changes (e.g.
hint: 'git pull ...') before pushing again.
hint: See the 'Note about fast-forwards' in 'git push --help' for details.
`,
			isBehind: true,
		},
		{
			name: "tip behind with odd casing",
			stderr: `error: failed to push some refs to 'git@github.com:lunarway/k8s-cluster-config.git'
hint: Updates were rejected because the TIP of your current branch is behind
hint: its remote counterpart. Integrate the remote changes (e.g.
hint: 'git pull ...') before pushing again.
hint: See the 'Note about fast-forwards' in 'git push --help' for details.
`,
			isBehind: true,
		},
		{
			name: "origin has work not available locally",
			stderr: `error: failed to push some refs to 'git@github.com:lunarway/k8s-cluster-config.git'
hint: Updates were rejected because the remote contains work that you do
hint: not have locally. This is usually caused by another repository pushing
hint: to the same ref. You may want to first integrate the remote changes
hint: (e.g., 'git pull ...') before pushing again.
hint: See the 'Note about fast-forwards' in 'git push --help' for details.
`,
			isBehind: true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			isBehind := isBranchBehindOrigin([]byte(tc.stderr))
			assert.Equal(t, tc.isBehind, isBehind, "result not as expected")
		})
	}
}

// TestService_pushMu_serializesConcurrentPushes verifies that pushMu serializes
// concurrent lock/unlock pairs — the same pattern used around gitPush calls in
// Commit and SignedCommit.
func TestService_pushMu_serializesConcurrentPushes(t *testing.T) {
	svc := &Service{}

	const workers = 5
	var (
		concurrent int64
		maxSeen    int64
		wg         sync.WaitGroup
	)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc.pushMu.Lock()
			cur := atomic.AddInt64(&concurrent, 1)
			if cur > atomic.LoadInt64(&maxSeen) {
				atomic.StoreInt64(&maxSeen, cur)
			}
			atomic.AddInt64(&concurrent, -1)
			svc.pushMu.Unlock()
		}()
	}
	wg.Wait()
	assert.Equal(t, int64(1), maxSeen, "at most one goroutine should hold the push mutex at a time")
}

// localRepositoryFixture is a clone of a bare origin repository. The checked
// out branch holds the commits first and pushed, which are pushed to origin,
// followed by unpushed, which only exists in the clone.
type localRepositoryFixture struct {
	repository LocalRepository
	originDir  string
	branch     string
	first      string
	pushed     string
	unpushed   string
}

func newLocalRepositoryFixture(t *testing.T) localRepositoryFixture {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)

	root := t.TempDir()
	f := localRepositoryFixture{
		repository: LocalRepository{Dir: filepath.Join(root, "work")},
		originDir:  filepath.Join(root, "origin"),
		branch:     "feature/wait",
	}
	runGit(t, root, "init", "--bare", "-b", "master", f.originDir)
	runGit(t, root, "clone", f.originDir, f.repository.Dir)
	configureIdentity(t, f.repository.Dir)
	runGit(t, f.repository.Dir, "checkout", "-b", f.branch)

	f.first = commitEmpty(t, f.repository.Dir, "first")
	f.pushed = commitEmpty(t, f.repository.Dir, "pushed")
	runGit(t, f.repository.Dir, "push", "origin", f.branch)
	f.unpushed = commitEmpty(t, f.repository.Dir, "unpushed")

	return f
}

// pushFromOtherClone pushes a new commit to the fixture branch on origin from
// another clone and returns its SHA.
func (f localRepositoryFixture) pushFromOtherClone(t *testing.T) string {
	t.Helper()
	otherDir := filepath.Join(t.TempDir(), "other")
	runGit(t, f.originDir, "clone", "--branch", f.branch, f.originDir, otherDir)
	configureIdentity(t, otherDir)
	sha := commitEmpty(t, otherDir, "from other clone")
	runGit(t, otherDir, "push", "origin", f.branch)

	return sha
}

func commitEmpty(t *testing.T, dir, message string) string {
	t.Helper()
	runGit(t, dir, "commit", "--allow-empty", "-m", message)

	return revParse(t, dir, "HEAD")
}

func TestLocalRepository(t *testing.T) {
	ctx := context.Background()
	f := newLocalRepositoryFixture(t)
	unknownSHA := strings.Repeat("0", 40)

	t.Run("HeadSHA", func(t *testing.T) {
		head, err := f.repository.HeadSHA(ctx)

		require.NoError(t, err)
		assert.Equal(t, f.unpushed, head)
	})

	t.Run("CommitExists", func(t *testing.T) {
		tt := []struct {
			name   string
			sha    string
			exists bool
		}{
			{name: "local commit", sha: f.unpushed, exists: true},
			{name: "abbreviated commit", sha: f.first[:7], exists: true},
			{name: "unknown commit", sha: unknownSHA, exists: false},
			{name: "option", sha: "--version", exists: false},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				assert.Equal(t, tc.exists, f.repository.CommitExists(ctx, tc.sha))
			})
		}
	})

	t.Run("IsAncestor", func(t *testing.T) {
		tt := []struct {
			name       string
			ancestor   string
			descendant string
			isAncestor bool
		}{
			{name: "ancestor", ancestor: f.first, descendant: f.unpushed, isAncestor: true},
			{name: "descendant", ancestor: f.unpushed, descendant: f.first, isAncestor: false},
			{name: "same commit", ancestor: f.pushed, descendant: f.pushed, isAncestor: true},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				isAncestor, err := f.repository.IsAncestor(ctx, tc.ancestor, tc.descendant)

				require.NoError(t, err)
				assert.Equal(t, tc.isAncestor, isAncestor)
			})
		}

		t.Run("unknown commit", func(t *testing.T) {
			_, err := f.repository.IsAncestor(ctx, unknownSHA, f.unpushed)

			assert.ErrorContains(t, err, unknownSHA)
		})
	})

	t.Run("IsPushed", func(t *testing.T) {
		tt := []struct {
			name   string
			sha    string
			pushed bool
		}{
			{name: "remote branch tip", sha: f.pushed, pushed: true},
			{name: "ancestor of remote branch tip", sha: f.first, pushed: true},
			{name: "local commit", sha: f.unpushed, pushed: false},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				pushed, err := f.repository.IsPushed(ctx, tc.sha, f.branch)

				require.NoError(t, err)
				assert.Equal(t, tc.pushed, pushed)
			})
		}

		t.Run("branch not on origin", func(t *testing.T) {
			_, err := f.repository.IsPushed(ctx, f.first, "not-pushed")

			assert.ErrorContains(t, err, "refs/remotes/origin/not-pushed")
		})
	})

	t.Run("CountCommits", func(t *testing.T) {
		tt := []struct {
			name  string
			from  string
			to    string
			count int
		}{
			{name: "ahead", from: f.first, to: f.unpushed, count: 2},
			{name: "behind", from: f.unpushed, to: f.first, count: 0},
			{name: "same commit", from: f.pushed, to: f.pushed, count: 0},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				count, err := f.repository.CountCommits(ctx, tc.from, tc.to)

				require.NoError(t, err)
				assert.Equal(t, tc.count, count)
			})
		}
	})

	t.Run("FetchBranch", func(t *testing.T) {
		sha := f.pushFromOtherClone(t)
		require.False(t, f.repository.CommitExists(ctx, sha), "commit must not exist before fetch")

		err := f.repository.FetchBranch(ctx, f.branch)

		require.NoError(t, err)
		assert.True(t, f.repository.CommitExists(ctx, sha), "commit must exist after fetch")
	})

	t.Run("FetchBranch branch not on origin", func(t *testing.T) {
		err := f.repository.FetchBranch(ctx, "not-pushed")

		assert.ErrorContains(t, err, "not-pushed")
	})
}
