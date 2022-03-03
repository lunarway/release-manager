package command_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/lunarway/release-manager/cmd/hamctl/command"
	"github.com/lunarway/release-manager/internal/artifact"
	internalhttp "github.com/lunarway/release-manager/internal/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelease(t *testing.T) {
	var (
		serviceName = "service-name"
		branch      = "master"
		artifactID  = "master-1-2"
	)
	t.Setenv("HAMCTL_USER_NAME", "test")
	t.Setenv("HAMCTL_USER_EMAIL", "test@example.com")

	// mocked manager server responding as configured in variables below
	var (
		foundArtifact   artifact.Spec
		releaseResponse func(r internalhttp.ReleaseRequest) (internalhttp.ReleaseResponse, *internalhttp.ErrorResponse)
	)
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "describe/latest-artifact"):
			resp := internalhttp.DescribeArtifactResponse{
				Artifacts: []artifact.Spec{
					foundArtifact,
				},
			}
			err := json.NewEncoder(rw).Encode(resp)
			require.NoError(t, err, "failed to encode test response payload")
		case strings.Contains(r.URL.Path, "release"):
			var req internalhttp.ReleaseRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err, "failed to dencode test request payload")

			resp, errorResponse := releaseResponse(req)
			if errorResponse != nil {
				internalhttp.Error(rw, errorResponse.Error(), errorResponse.Status)
				return
			}
			err = json.NewEncoder(rw).Encode(resp)
			require.NoError(t, err, "failed to encode test response payload")
		default:
			rw.WriteHeader(http.StatusInternalServerError)
		}
	}))

	c := internalhttp.Client{
		BaseURL: server.URL,
	}

	runCommand := func(t *testing.T, args ...string) []string {
		var output []string

		cmd := command.NewRelease(&c, &serviceName, func(f string, args ...interface{}) {
			output = append(output, fmt.Sprintf(f, args...))
		})

		cmd.SetArgs(args)

		err := cmd.Execute()
		require.NoError(t, err, "unexpected execution error")

		// mask GUID to make output assertable
		output = maskGUID(output)

		return output
	}

	t.Run("multiple environments", func(t *testing.T) {
		foundArtifact = artifact.Spec{
			ID:      artifactID,
			Service: serviceName,
		}
		releaseResponse = func(req internalhttp.ReleaseRequest) (internalhttp.ReleaseResponse, *internalhttp.ErrorResponse) {
			return internalhttp.ReleaseResponse{
				Service:       serviceName,
				ToEnvironment: req.Environment,
				Tag:           artifactID,
			}, nil
		}

		output := runCommand(t, "--branch", branch, "--env", "dev,prod")

		assert.Equal(t, []string{
			"Release of service service-name using branch master\n",
			"[✓] Release of master-1-2 to dev initialized\n",
			"[✓] Release of master-1-2 to prod initialized\n",
		}, output)
	})

	t.Run("environment up to date", func(t *testing.T) {
		foundArtifact = artifact.Spec{
			ID:      artifactID,
			Service: serviceName,
		}
		releaseResponse = func(req internalhttp.ReleaseRequest) (internalhttp.ReleaseResponse, *internalhttp.ErrorResponse) {
			resp := internalhttp.ReleaseResponse{
				Service:       serviceName,
				ToEnvironment: req.Environment,
				Tag:           artifactID,
			}
			if req.Environment == "prod" {
				resp.Status = "Environment prod is already up-to-date"
			}
			return resp, nil
		}

		output := runCommand(t, "--branch", branch, "--env", "dev,prod")

		assert.Equal(t, []string{
			"Release of service service-name using branch master\n",
			"[✓] Release of master-1-2 to dev initialized\n",
			"[✓] Environment prod is already up-to-date\n",
		}, output)
	})

	t.Run("unknown environment for single env", func(t *testing.T) {
		foundArtifact = artifact.Spec{
			ID:      artifactID,
			Service: serviceName,
		}
		releaseResponse = func(req internalhttp.ReleaseRequest) (internalhttp.ReleaseResponse, *internalhttp.ErrorResponse) {
			return internalhttp.ReleaseResponse{}, &internalhttp.ErrorResponse{
				Status:  http.StatusBadRequest,
				Message: "cannot release master-1-2 to environment dev due to branch restriction policy",
			}
		}

		output := runCommand(t, "--branch", branch, "--env", "dev")

		assert.Equal(t, []string{
			"Release of service service-name using branch master\n",
			"[X] cannot release master-1-2 to environment dev due to branch restriction policy (reference: GUID)\n",
		}, output)
	})

	t.Run("unknown environment", func(t *testing.T) {
		foundArtifact = artifact.Spec{
			ID:      artifactID,
			Service: serviceName,
		}
		releaseResponse = func(req internalhttp.ReleaseRequest) (internalhttp.ReleaseResponse, *internalhttp.ErrorResponse) {
			resp := internalhttp.ReleaseResponse{
				Service:       serviceName,
				ToEnvironment: req.Environment,
				Tag:           artifactID,
			}
			if req.Environment == "dev" {
				return resp, &internalhttp.ErrorResponse{
					Status:  http.StatusBadRequest,
					Message: "cannot release master-1-2 to environment dev due to branch restriction policy",
				}
			}
			return resp, nil
		}

		output := runCommand(t, "--branch", branch, "--env", "dev,prod")

		assert.Equal(t, []string{
			"Release of service service-name using branch master\n",
			"[X] cannot release master-1-2 to environment dev due to branch restriction policy (reference: GUID)\n",
			"[✓] Release of master-1-2 to prod initialized\n",
		}, output)
	})
}

// maskGUID masks any guid with the text GUID.
func maskGUID(output []string) []string {
	var masked []string
	r := regexp.MustCompile(".{8}-.{4}-.{4}-.{4}-.{12}")
	for _, line := range output {
		m := r.ReplaceAllString(line, "GUID")
		masked = append(masked, m)
	}
	return masked
}

func TestRelease_emptyEnvValue(t *testing.T) {
	serviceName := "service-name"
	c := internalhttp.Client{}

	cmd := command.NewRelease(&c, &serviceName, func(f string, args ...interface{}) {
		t.Logf(f, args...)
	})

	cmd.SetArgs([]string{"--env", ""})

	err := cmd.Execute()
	require.Error(t, err, "unexpected execution error")
}
