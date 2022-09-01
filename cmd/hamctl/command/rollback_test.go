package command_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lunarway/release-manager/cmd/hamctl/command"
	"github.com/lunarway/release-manager/cmd/hamctl/command/actions"
	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/git"
	internalhttp "github.com/lunarway/release-manager/internal/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRollback(t *testing.T) {
	var (
		serviceName = "some-service-name"
		env         = "dev"
	)
	var describeReleaseResponse func() []internalhttp.DescribeReleaseResponseRelease

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var releaseRequest internalhttp.ReleaseRequest
		_ = json.NewDecoder(r.Body).Decode(&releaseRequest)

		switch {

		case strings.Contains(r.URL.Path, "describe/release/"):
			resp := internalhttp.DescribeReleaseResponse{
				Service:     serviceName,
				Environment: env,
				Releases:    describeReleaseResponse(),
			}
			err := json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)
		case strings.Contains(r.URL.Path, "release") && r.Method == http.MethodPost && releaseRequest.ArtifactID == "artifact-1":
			resp := internalhttp.ReleaseResponse{
				Service:       serviceName,
				ReleaseID:     "some-release-id",
				Status:        "released",
				ToEnvironment: env,
				Tag:           "dev",
			}

			err := json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)

		case strings.Contains(r.URL.Path, "release") && r.Method == http.MethodPost && releaseRequest.ArtifactID == "artifact-0":
			resp := internalhttp.ReleaseResponse{
				Service:       serviceName,
				ReleaseID:     "some-release-id",
				Status:        "released",
				ToEnvironment: env,
				Tag:           "dev",
			}

			err := json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)

		default:
			require.Fail(t, "http url was not found")
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	c := internalhttp.Client{
		BaseURL: server.URL,
	}

	gitConfigAPI := git.GitConfigAPIMock{
		CommitterDetailsFunc: func() (string, string, error) {
			return "name", "email", nil
		},
	}

	releaseClient := actions.NewReleaseHttpClient(&gitConfigAPI, &c)

	runCommand := func(t *testing.T, selectRollback command.SelectRollbackRelease, args ...string) ([]string, error) {
		var output []string
		cmd := command.NewRollback(&c, &serviceName, func(f string, args ...interface{}) {
			output = append(output, fmt.Sprintf(f, args...))
		}, selectRollback, releaseClient)
		cmd.SetArgs(args)
		err := cmd.Execute()

		return output, err
	}

	t.Run("test rollback to previous commit with no artifacts specified", func(t *testing.T) {
		describeReleaseResponse = provideArtifacts(3)

		var rollback command.SelectRollbackRelease = func(environment string, releases []internalhttp.DescribeReleaseResponseRelease) (int, error) {
			return 1, nil
		}

		output, err := runCommand(t, rollback, "--env", "dev")

		require.NoError(t, err)
		assert.Equal(
			t,
			[]string{"[✓] Starting rollback of service some-service-name to artifact-1\n", "[✓] released\n"},
			output,
		)
	})

	t.Run("test rollback to previous commit with no artifacts specified select index 0", func(t *testing.T) {
		describeReleaseResponse = provideArtifacts(3)
		var rollback command.SelectRollbackRelease = func(environment string, releases []internalhttp.DescribeReleaseResponseRelease) (int, error) {
			return 0, nil
		}

		output, err := runCommand(t, rollback, "--env", "dev")

		require.NoError(t, err)
		assert.Equal(
			t,
			[]string{"[✓] Starting rollback of service some-service-name to artifact-0\n", "[✓] released\n"},
			output,
		)
	})

	t.Run("test rollback to previous commit with no artifacts not enough releases", func(t *testing.T) {
		describeReleaseResponse = provideArtifacts(1)
		var rollback command.SelectRollbackRelease = func(environment string, releases []internalhttp.DescribeReleaseResponseRelease) (int, error) {
			return 0, nil
		}

		_, err := runCommand(t, rollback, "--env", "dev")

		require.ErrorContains(t, err, "can't do rollback")
	})

	t.Run("test rollback to previous commit with artifact specified", func(t *testing.T) {
		describeReleaseResponse = provideArtifacts(3)
		var rollback command.SelectRollbackRelease = func(environment string, releases []internalhttp.DescribeReleaseResponseRelease) (int, error) {
			return -1, nil
		}

		output, err := runCommand(t, rollback, "--env", "dev", "--artifact", "artifact-0")

		require.NoError(t, err)
		assert.Equal(
			t,
			[]string{"[✓] Starting rollback of service some-service-name to artifact-0\n", "[✓] released\n"},
			output,
		)
	})

	t.Run("test rollback to previous commit with artifact specified, but not found", func(t *testing.T) {
		describeReleaseResponse = provideArtifacts(3)
		var rollback command.SelectRollbackRelease = func(environment string, releases []internalhttp.DescribeReleaseResponseRelease) (int, error) {
			return -1, nil
		}

		_, err := runCommand(t, rollback, "--env", "dev", "--artifact", "artifact-38")

		require.ErrorContains(t, err, "isn't found in the last 10")
	})
}

func provideArtifacts(amount int) func() []internalhttp.DescribeReleaseResponseRelease {
	var releases = make([]internalhttp.DescribeReleaseResponseRelease, amount)

	for i := 0; i < amount; i++ {
		releases[i] = internalhttp.DescribeReleaseResponseRelease{
			ReleaseIndex: i,
			Artifact: artifact.Spec{
				ID: fmt.Sprintf("artifact-%d", i),
			},
		}
	}

	return func() []internalhttp.DescribeReleaseResponseRelease {
		return releases
	}
}
