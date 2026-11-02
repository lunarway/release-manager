package command_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lunarway/release-manager/cmd/hamctl/command"
	"github.com/lunarway/release-manager/cmd/hamctl/command/actions"
	"github.com/lunarway/release-manager/internal/artifact"
	internalhttp "github.com/lunarway/release-manager/internal/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testService = "service-name"
	// testBranch is the current branch. Its artifacts record it as
	// testArtifactBranch.
	testBranch         = "feature/wait"
	testArtifactBranch = "feature_wait"

	devEnv        = "dev"
	prodEnv       = "prod"
	devAndProdEnv = devEnv + "," + prodEnv

	// fastPoll makes waiting commands poll without noticeable delay.
	fastPoll = time.Millisecond
	// slowPoll makes waiting commands poll only once before a short
	// --wait-timeout expires.
	slowPoll = time.Hour
)

var (
	commitA = strings.Repeat("a", 40)
	commitB = strings.Repeat("b", 40)
	commitC = strings.Repeat("c", 40)
	commitD = strings.Repeat("d", 40)
	// commitE is on a history of testBranch on origin that HEAD is not part of.
	commitE = strings.Repeat("e", 40)
	// commitF is not on any branch.
	commitF = strings.Repeat("f", 40)

	artifactA = branchArtifact(commitA)
	artifactB = branchArtifact(commitB)
	artifactC = branchArtifact(commitC)
	artifactD = branchArtifact(commitD)
	artifactE = branchArtifact(commitE)
	artifactF = branchArtifact(commitF)
)

// branchArtifact returns an artifact of testBranch built from commit.
func branchArtifact(commit string) artifact.Spec {
	return artifact.Spec{
		ID:      fmt.Sprintf("%s-%s-0123456789", testArtifactBranch, commit[:10]),
		Service: testService,
		Application: artifact.Repository{
			Branch: testArtifactBranch,
			SHA:    commit,
		},
	}
}

type NoopAuthClient struct{}

func (NoopAuthClient) Access(context.Context) (*http.Client, error) {
	return &http.Client{}, nil
}

// branchState is the state of testBranch on the release manager when it
// receives a describe/artifact request.
type branchState struct {
	// artifacts of the branch, newest first.
	artifacts []artifact.Spec
	// failure is the message the request fails with, if set.
	failure string
}

func artifacts(specs ...artifact.Spec) branchState {
	return branchState{artifacts: specs}
}

func failure(message string) branchState {
	return branchState{failure: message}
}

// fakeReleaseManager is a release manager server.
type fakeReleaseManager struct {
	// latestArtifact is served by describe/latest-artifact.
	latestArtifact artifact.Spec
	// branch is served to describe/artifact requests in turn. Requests after
	// the last state are served the last state.
	branch []branchState
	// releaseFailures maps environments to the message releases to them fail
	// with.
	releaseFailures map[string]string
	// releaseStatus is the status of successful releases.
	releaseStatus map[string]string

	mu sync.Mutex
	// describeArtifactQueries are the raw queries of the describe/artifact
	// requests received.
	describeArtifactQueries []string
}

func (m *fakeReleaseManager) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/describe/latest-artifact/"):
		encodeJSON(rw, internalhttp.DescribeArtifactResponse{Artifacts: []artifact.Spec{m.latestArtifact}})
	case strings.HasPrefix(r.URL.Path, "/describe/artifact/"):
		m.describeArtifact(rw, r.URL.Query())
	case r.URL.Path == "/release":
		m.release(rw, r)
	default:
		rw.WriteHeader(http.StatusNotFound)
	}
}

func (m *fakeReleaseManager) describeArtifact(rw http.ResponseWriter, query url.Values) {
	m.mu.Lock()
	m.describeArtifactQueries = append(m.describeArtifactQueries, query.Encode())
	state := m.branch[min(len(m.describeArtifactQueries), len(m.branch))-1]
	m.mu.Unlock()

	if state.failure != "" {
		internalhttp.Error(rw, state.failure, http.StatusInternalServerError)
		return
	}
	count, err := strconv.Atoi(query.Get("count"))
	if err != nil {
		internalhttp.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	encodeJSON(rw, internalhttp.DescribeArtifactResponse{
		Service:   testService,
		Artifacts: state.artifacts[:min(count, len(state.artifacts))],
	})
}

func (m *fakeReleaseManager) release(rw http.ResponseWriter, r *http.Request) {
	var req internalhttp.ReleaseRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		internalhttp.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	if message, ok := m.releaseFailures[req.Environment]; ok {
		internalhttp.Error(rw, message, http.StatusBadRequest)
		return
	}
	encodeJSON(rw, internalhttp.ReleaseResponse{
		Service:       req.Service,
		ToEnvironment: req.Environment,
		Tag:           req.ArtifactID,
		Status:        m.releaseStatus[req.Environment],
	})
}

func (m *fakeReleaseManager) queries() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.describeArtifactQueries
}

func encodeJSON(rw http.ResponseWriter, response any) {
	err := json.NewEncoder(rw).Encode(response)
	if err != nil {
		internalhttp.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

// fakeRepository is a Git repository with linear history checked out on
// testBranch.
type fakeRepository struct {
	head string
	// parents maps the commits in the repository to their parent. The root
	// commit maps to "".
	parents map[string]string
	// originParents maps the commits only on origin to their parent. Fetching
	// adds them to parents.
	originParents map[string]string
	// remoteBranches maps branches to the tip of their remote-tracking branch.
	remoteBranches map[string]string
	fetchErr       error
	// fetches are the branches fetched.
	fetches []string
}

// newFakeRepository returns a repository with HEAD at the last of commits,
// which form the history of testBranch, oldest first, and are pushed to
// origin.
func newFakeRepository(commits ...string) *fakeRepository {
	r := &fakeRepository{
		parents:        map[string]string{},
		originParents:  map[string]string{},
		remoteBranches: map[string]string{},
	}
	for _, commit := range commits {
		r.commit(commit)
	}
	r.remoteBranches[testBranch] = r.head

	return r
}

func (r *fakeRepository) HeadSHA(context.Context) (string, error) {
	return r.head, nil
}

func (r *fakeRepository) CommitExists(_ context.Context, sha string) bool {
	_, ok := r.parents[sha]

	return ok
}

func (r *fakeRepository) FetchBranch(_ context.Context, branch string) error {
	r.fetches = append(r.fetches, branch)
	if r.fetchErr != nil {
		return r.fetchErr
	}
	maps.Copy(r.parents, r.originParents)

	return nil
}

func (r *fakeRepository) IsAncestor(ctx context.Context, ancestor, descendant string) (bool, error) {
	if !r.CommitExists(ctx, ancestor) || !r.CommitExists(ctx, descendant) {
		return false, fmt.Errorf("not a valid commit: %s or %s", ancestor, descendant)
	}
	for commit := descendant; commit != ""; commit = r.parents[commit] {
		if commit == ancestor {
			return true, nil
		}
	}

	return false, nil
}

func (r *fakeRepository) IsPushed(ctx context.Context, sha, branch string) (bool, error) {
	tip, ok := r.remoteBranches[branch]
	if !ok {
		return false, fmt.Errorf("not a valid object name refs/remotes/origin/%s", branch)
	}

	return r.IsAncestor(ctx, sha, tip)
}

func (r *fakeRepository) CountCommits(_ context.Context, from, to string) (int, error) {
	count := 0
	for commit := to; commit != from; commit = r.parents[commit] {
		count++
	}

	return count, nil
}

// commit commits on top of HEAD without pushing.
func (r *fakeRepository) commit(sha string) *fakeRepository {
	r.parents[sha] = r.head
	r.head = sha

	return r
}

// resetHard moves HEAD to sha.
func (r *fakeRepository) resetHard(sha string) *fakeRepository {
	r.head = sha

	return r
}

// pushedByOthers adds commits to origin, oldest first, on top of parent.
func (r *fakeRepository) pushedByOthers(parent string, commits ...string) *fakeRepository {
	for _, commit := range commits {
		r.originParents[commit] = parent
		parent = commit
	}

	return r
}

func (r *fakeRepository) withoutRemoteBranch() *fakeRepository {
	delete(r.remoteBranches, testBranch)

	return r
}

func (r *fakeRepository) failingFetch(err error) *fakeRepository {
	r.fetchErr = err

	return r
}

// runRelease runs the release command with flags to environments against
// server in repository with the current branch testBranch and returns its
// output with GUIDs masked.
func runRelease(
	t *testing.T,
	server *fakeReleaseManager,
	repository command.GitRepository,
	pollInterval time.Duration,
	environments string,
	flags ...string,
) ([]string, error) {
	t.Helper()
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)

	client := internalhttp.Client{
		BaseURL: httpServer.URL,
		Auth:    NoopAuthClient{},
	}
	service := testService
	var output []string
	cmd := command.NewRelease(&client, &service, func(f string, args ...any) {
		output = append(output, maskGUID(fmt.Sprintf(f, args...)))
	}, actions.NewReleaseHttpClient(&client), func() string { return testBranch }, repository, pollInterval)
	cmd.SetArgs(append([]string{"--env", environments}, flags...))
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err := cmd.Execute()

	return output, err
}

// releasedTo returns the output of releasing spec to env.
func releasedTo(spec artifact.Spec, env string) string {
	return fmt.Sprintf("[✓] Release of %s to %s initialized\n", spec.ID, env)
}

// releasingBranch returns the output announcing a release from branch.
func releasingBranch(branch string) string {
	return fmt.Sprintf("Release of service %s using branch %s\n", testService, branch)
}

func TestRelease(t *testing.T) {
	t.Setenv("HAMCTL_USER_NAME", "test")
	t.Setenv("HAMCTL_USER_EMAIL", "test@example.com")
	masterArtifact := artifact.Spec{ID: "master-1-2", Service: testService}
	branchRestricted := "cannot release master-1-2 to environment dev due to branch restriction policy"
	releaseMaster := []string{"--branch", "master"}

	tt := []struct {
		name         string
		server       *fakeReleaseManager
		environments string
		flags        []string
		output       []string
	}{
		{
			name:         "multiple environments",
			server:       &fakeReleaseManager{latestArtifact: masterArtifact},
			environments: devAndProdEnv,
			flags:        releaseMaster,
			output: []string{
				releasingBranch("master"),
				releasedTo(masterArtifact, devEnv),
				releasedTo(masterArtifact, prodEnv),
			},
		},
		{
			name: "environment up to date",
			server: &fakeReleaseManager{
				latestArtifact: masterArtifact,
				releaseStatus:  map[string]string{prodEnv: "Environment prod is already up-to-date"},
			},
			environments: devAndProdEnv,
			flags:        releaseMaster,
			output: []string{
				releasingBranch("master"),
				releasedTo(masterArtifact, devEnv),
				"[✓] Environment prod is already up-to-date\n",
			},
		},
		{
			name: "unknown environment for single env",
			server: &fakeReleaseManager{
				latestArtifact:  masterArtifact,
				releaseFailures: map[string]string{devEnv: branchRestricted},
			},
			environments: devEnv,
			flags:        releaseMaster,
			output: []string{
				releasingBranch("master"),
				"[X] " + branchRestricted + " (reference: GUID)\n",
			},
		},
		{
			name: "unknown environment",
			server: &fakeReleaseManager{
				latestArtifact:  masterArtifact,
				releaseFailures: map[string]string{devEnv: branchRestricted},
			},
			environments: devAndProdEnv,
			flags:        releaseMaster,
			output: []string{
				releasingBranch("master"),
				"[X] " + branchRestricted + " (reference: GUID)\n",
				releasedTo(masterArtifact, prodEnv),
			},
		},
		{
			name:         "artifact",
			server:       &fakeReleaseManager{},
			environments: devEnv,
			flags:        []string{"--artifact", masterArtifact.ID},
			output: []string{
				"Release of service: service-name\n",
				releasedTo(masterArtifact, devEnv),
			},
		},
		{
			name:         "current git branch",
			server:       &fakeReleaseManager{latestArtifact: artifactC},
			environments: devEnv,
			flags:        []string{"--current-branch"},
			output: []string{
				releasingBranch(testBranch),
				releasedTo(artifactC, devEnv),
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			output, err := runRelease(t, tc.server, newFakeRepository(commitA, commitB, commitC), fastPoll,
				tc.environments, tc.flags...)

			require.NoError(t, err)
			assert.Equal(t, tc.output, output)
		})
	}
}

func TestRelease_currentBranchWithoutWait(t *testing.T) {
	t.Setenv("HAMCTL_USER_NAME", "test")
	t.Setenv("HAMCTL_USER_EMAIL", "test@example.com")

	tt := []struct {
		name       string
		repository *fakeRepository
		latest     artifact.Spec
		// warning is the output before the release, if any.
		warning string
		fetches int
	}{
		{
			name:       "latest built from HEAD",
			repository: newFakeRepository(commitA, commitB),
			latest:     artifactB,
		},
		{
			name:       "latest older than HEAD",
			repository: newFakeRepository(commitA, commitB),
			latest:     artifactA,
			warning: "Latest artifact feature_wait-aaaaaaaaaa-0123456789 is built from aaaaaaa, not HEAD bbbbbbb. " +
				"CI may still be building, or HEAD was built earlier; use --wait.\n",
		},
		{
			name:       "latest unknown after fetch",
			repository: newFakeRepository(commitA, commitB),
			latest:     artifactF,
			warning: "Latest artifact feature_wait-ffffffffff-0123456789 is built from fffffff, not HEAD bbbbbbb. " +
				"CI may still be building, or HEAD was built earlier; use --wait.\n",
			fetches: 1,
		},
		{
			name:       "latest on other history",
			repository: newFakeRepository(commitA, commitB).pushedByOthers(commitA, commitE),
			latest:     artifactE,
			warning: "Latest artifact feature_wait-eeeeeeeeee-0123456789 is built from eeeeeee, not HEAD bbbbbbb. " +
				"CI may still be building, or HEAD was built earlier; use --wait.\n",
			fetches: 1,
		},
		{
			name:       "latest newer than HEAD",
			repository: newFakeRepository(commitA, commitB).pushedByOthers(commitB, commitC),
			latest:     artifactC,
			warning: "Releasing feature_wait-cccccccccc-0123456789 built from ccccccc, " +
				"which is 1 commit ahead of HEAD bbbbbbb\n",
			fetches: 1,
		},
		{
			name:       "comparison with HEAD fails",
			repository: newFakeRepository(commitA, commitB).failingFetch(errors.New("network unreachable")),
			latest:     artifactF,
			warning:    "Could not compare latest artifact feature_wait-ffffffffff-0123456789 with HEAD: network unreachable\n",
			fetches:    1,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			server := &fakeReleaseManager{latestArtifact: tc.latest}

			output, err := runRelease(t, server, tc.repository, fastPoll, devEnv, "--current-branch")

			require.NoError(t, err)
			expected := []string{releasingBranch(testBranch), releasedTo(tc.latest, devEnv)}
			if tc.warning != "" {
				expected = append([]string{tc.warning}, expected...)
			}
			assert.Equal(t, expected, output)
			assert.Len(t, tc.repository.fetches, tc.fetches, "number of fetches")
			assert.Empty(t, server.queries(), "describe/artifact must not be requested")
		})
	}
}

func TestRelease_currentBranchWait(t *testing.T) {
	t.Setenv("HAMCTL_USER_NAME", "test")
	t.Setenv("HAMCTL_USER_EMAIL", "test@example.com")
	const (
		recentQuery = "branch=feature_wait&count=20"
		latestQuery = "branch=feature_wait&count=1"
	)

	tt := []struct {
		name       string
		repository *fakeRepository
		branch     []branchState
		// environments are released to, dev if empty.
		environments string
		// flags are passed in addition to --current-branch --wait.
		flags        []string
		pollInterval time.Duration
		output       []string
		err          string
		queries      []string
		fetches      int
	}{
		{
			name:         "artifact of HEAD is latest",
			repository:   newFakeRepository(commitA, commitB, commitC),
			branch:       []branchState{artifacts(artifactC, artifactB, artifactA)},
			environments: devAndProdEnv,
			pollInterval: fastPoll,
			output: []string{
				releasingBranch(testBranch),
				releasedTo(artifactC, devEnv),
				releasedTo(artifactC, prodEnv),
			},
			queries: []string{recentQuery},
		},
		{
			name:         "artifact of HEAD is older than latest",
			repository:   newFakeRepository(commitA, commitB, commitC).resetHard(commitB),
			branch:       []branchState{artifacts(artifactC, artifactB, artifactA)},
			pollInterval: fastPoll,
			output: []string{
				"Releasing feature_wait-bbbbbbbbbb-0123456789 built from HEAD bbbbbbb; " +
					"newer artifact feature_wait-cccccccccc-0123456789 exists on feature/wait\n",
				releasingBranch(testBranch),
				releasedTo(artifactB, devEnv),
			},
			queries: []string{recentQuery},
		},
		{
			name:       "artifact of HEAD is latest on first poll",
			repository: newFakeRepository(commitA, commitB, commitC),
			branch: []branchState{
				artifacts(artifactB, artifactA),
				artifacts(artifactC, artifactB, artifactA),
			},
			pollInterval: fastPoll,
			output: []string{
				releasingBranch(testBranch),
				releasedTo(artifactC, devEnv),
			},
			queries: []string{recentQuery, latestQuery},
		},
		{
			name:       "latest is older than HEAD until artifact of HEAD is latest",
			repository: newFakeRepository(commitA, commitB, commitC),
			branch: []branchState{
				artifacts(artifactB, artifactA),
				artifacts(artifactB, artifactA),
				artifacts(artifactB, artifactA),
				artifacts(artifactC, artifactB, artifactA),
			},
			pollInterval: fastPoll,
			output: []string{
				"Waiting for artifact of ccccccc (latest: feature_wait-bbbbbbbbbb-0123456789)\n",
				releasingBranch(testBranch),
				releasedTo(artifactC, devEnv),
			},
			queries: []string{recentQuery, latestQuery, latestQuery, latestQuery},
		},
		{
			name:       "no artifacts until artifact of HEAD",
			repository: newFakeRepository(commitA),
			branch: []branchState{
				artifacts(),
				artifacts(),
				artifacts(artifactA),
			},
			pollInterval: fastPoll,
			output: []string{
				"Waiting for artifact of aaaaaaa (latest: none)\n",
				releasingBranch(testBranch),
				releasedTo(artifactA, devEnv),
			},
			queries: []string{recentQuery, latestQuery, latestQuery},
		},
		{
			name:       "latest changes while waiting",
			repository: newFakeRepository(commitA, commitB, commitC),
			branch: []branchState{
				artifacts(),
				artifacts(artifactA),
				artifacts(artifactA),
				artifacts(artifactB, artifactA),
				artifacts(artifactC, artifactB, artifactA),
			},
			pollInterval: fastPoll,
			output: []string{
				"Waiting for artifact of ccccccc (latest: feature_wait-aaaaaaaaaa-0123456789)\n",
				"Waiting for artifact of ccccccc (latest: feature_wait-bbbbbbbbbb-0123456789)\n",
				releasingBranch(testBranch),
				releasedTo(artifactC, devEnv),
			},
			queries: []string{recentQuery, latestQuery, latestQuery, latestQuery, latestQuery},
		},
		{
			name:         "latest is one commit newer than HEAD",
			repository:   newFakeRepository(commitA, commitB, commitC).resetHard(commitB),
			branch:       []branchState{artifacts(artifactC, artifactA)},
			pollInterval: fastPoll,
			output: []string{
				"Releasing feature_wait-cccccccccc-0123456789 built from ccccccc, which is 1 commit ahead of HEAD bbbbbbb\n",
				releasingBranch(testBranch),
				releasedTo(artifactC, devEnv),
			},
			queries: []string{recentQuery, latestQuery},
		},
		{
			name:         "latest is newer than HEAD on origin",
			repository:   newFakeRepository(commitA, commitB).pushedByOthers(commitB, commitC, commitD),
			branch:       []branchState{artifacts(artifactD, artifactA)},
			pollInterval: fastPoll,
			output: []string{
				"Releasing feature_wait-dddddddddd-0123456789 built from ddddddd, which is 2 commits ahead of HEAD bbbbbbb\n",
				releasingBranch(testBranch),
				releasedTo(artifactD, devEnv),
			},
			queries: []string{recentQuery, latestQuery},
			fetches: 1,
		},
		{
			name:       "latest is unknown until artifact of HEAD is latest",
			repository: newFakeRepository(commitA, commitB, commitC),
			branch: []branchState{
				artifacts(artifactF),
				artifacts(artifactF),
				artifacts(artifactF),
				artifacts(artifactC, artifactF),
			},
			pollInterval: fastPoll,
			output: []string{
				"Waiting for artifact of ccccccc (latest: feature_wait-ffffffffff-0123456789)\n",
				releasingBranch(testBranch),
				releasedTo(artifactC, devEnv),
			},
			queries: []string{recentQuery, latestQuery, latestQuery, latestQuery},
			fetches: 1,
		},
		{
			name:         "latest on other history until timeout",
			repository:   newFakeRepository(commitA, commitB, commitC).pushedByOthers(commitA, commitE),
			branch:       []branchState{artifacts(artifactE)},
			flags:        []string{"--wait-timeout", "1ms"},
			pollInterval: slowPoll,
			output: []string{
				"Waiting for artifact of ccccccc (latest: feature_wait-eeeeeeeeee-0123456789)\n",
			},
			err: "Timed out after 1ms waiting for artifact of ccccccc on feature/wait; " +
				"latest is feature_wait-eeeeeeeeee-0123456789. Check CI.",
			queries: []string{recentQuery, latestQuery},
			fetches: 1,
		},
		{
			name:         "no artifacts until timeout",
			repository:   newFakeRepository(commitA),
			branch:       []branchState{artifacts()},
			flags:        []string{"--wait-timeout", "1ms"},
			pollInterval: slowPoll,
			output: []string{
				"Waiting for artifact of aaaaaaa (latest: none)\n",
			},
			err:     "Timed out after 1ms waiting for artifact of aaaaaaa on feature/wait; latest is none. Check CI.",
			queries: []string{recentQuery, latestQuery},
		},
		{
			name:         "HEAD not pushed",
			repository:   newFakeRepository(commitA, commitB).commit(commitC),
			branch:       []branchState{artifacts(artifactB, artifactA)},
			pollInterval: fastPoll,
			err:          "HEAD ccccccc is not pushed to origin/feature/wait",
		},
		{
			name:         "branch not on origin",
			repository:   newFakeRepository(commitA).withoutRemoteBranch(),
			branch:       []branchState{artifacts(artifactA)},
			pollInterval: fastPoll,
			err:          "HEAD aaaaaaa is not pushed to origin/feature/wait",
		},
		{
			name:         "looking up recent artifacts fails",
			repository:   newFakeRepository(commitA, commitB),
			branch:       []branchState{failure("storage unavailable")},
			pollInterval: fastPoll,
			err:          "storage unavailable",
			queries:      []string{recentQuery},
		},
		{
			name:       "polling latest artifact fails",
			repository: newFakeRepository(commitA, commitB),
			branch: []branchState{
				artifacts(artifactA),
				artifacts(artifactA),
				failure("storage unavailable"),
			},
			pollInterval: fastPoll,
			output: []string{
				"Waiting for artifact of bbbbbbb (latest: feature_wait-aaaaaaaaaa-0123456789)\n",
			},
			err:     "storage unavailable",
			queries: []string{recentQuery, latestQuery, latestQuery},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			server := &fakeReleaseManager{branch: tc.branch}
			environments := tc.environments
			if environments == "" {
				environments = devEnv
			}
			flags := append([]string{"--current-branch", "--wait"}, tc.flags...)

			output, err := runRelease(t, server, tc.repository, tc.pollInterval, environments, flags...)

			if tc.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.err)
			}
			assert.Equal(t, tc.output, output)
			assert.Equal(t, tc.queries, server.queries(), "describe/artifact queries")
			assert.Len(t, tc.repository.fetches, tc.fetches, "number of fetches")
		})
	}
}

func TestRelease_emptyEnvValue(t *testing.T) {
	server := &fakeReleaseManager{latestArtifact: artifactA}

	output, err := runRelease(t, server, newFakeRepository(commitA), fastPoll, "", "--current-branch")

	require.EqualError(t, err, "--env must contain at least one value")
	assert.Empty(t, output)
}

func TestRelease_waitRequiresCurrentBranch(t *testing.T) {
	tt := map[string][]string{
		"without branch": nil,
		"with branch":    {"--branch", "master"},
		"with artifact":  {"--artifact", "master-1-2"},
	}
	for name, flags := range tt {
		t.Run(name, func(t *testing.T) {
			server := &fakeReleaseManager{latestArtifact: artifactA, branch: []branchState{artifacts(artifactA)}}

			output, err := runRelease(t, server, newFakeRepository(commitA), fastPoll, devEnv, append(flags, "--wait")...)

			require.EqualError(t, err, "--wait requires --current-branch")
			assert.Empty(t, output)
			assert.Empty(t, server.queries())
		})
	}
}

// guid matches GUIDs, e.g. the references of error responses.
var guid = regexp.MustCompile(".{8}-.{4}-.{4}-.{4}-.{12}")

// maskGUID masks any guid with the text GUID.
func maskGUID(line string) string {
	return guid.ReplaceAllString(line, "GUID")
}
