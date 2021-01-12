package git

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/stretchr/testify/assert"
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

// TestLocateServiceReleaseRollbackSkip tests that
// LocateServiceReleaseRollbackSkip works with multiple lookups on the same
// repository where each found release is checked out on the repo.
func TestLocateServiceReleaseRollbackSkip(t *testing.T) {
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

	s := Service{
		Tracer: tracing.NewNoop(),
	}

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
			t.Fatalf("failed to checkout: %v", err)
		}
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
