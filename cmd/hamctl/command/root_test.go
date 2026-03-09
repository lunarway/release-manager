package command

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultShuttleString(t *testing.T) {
	tt := []struct {
		name      string
		flagValue string
		spec      shuttleSpec
		output    string
	}{
		{
			name:      "flag value",
			flagValue: "value",
			spec: shuttleSpec{
				Vars: shuttleSpecVars{
					Service: "service",
				},
			},
			output: "value",
		},
		{
			name:      "empty flag value",
			flagValue: "",
			spec: shuttleSpec{
				Vars: shuttleSpecVars{
					Service: "service",
				},
			},
			output: "service",
		},
		{
			name:      "whitespace flag value",
			flagValue: "   ",
			spec: shuttleSpec{
				Vars: shuttleSpecVars{
					Service: "service",
				},
			},
			output: "service",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			defaultShuttleString(func() (shuttleSpec, bool) {
				return tc.spec, true
			}, &tc.flagValue, func(s *shuttleSpec) string {
				return s.Vars.Service
			})
			assert.Equal(t, tc.output, tc.flagValue, "flag value not as expected")
		})
	}
}

func TestDefaultShuttleString_noSpec(t *testing.T) {
	var flagValue string
	defaultShuttleString(func() (shuttleSpec, bool) {
		return shuttleSpec{}, false
	}, &flagValue, func(s *shuttleSpec) string {
		return s.Vars.Service
	})
	assert.Equal(t, "", flagValue, "flag value not as expected")
}

func TestRoot_RequiredInputs(t *testing.T) {
	type input struct {
		service     string
		httpBaseURL string

		url           string
		oauthClientID string
		oauthIDPURL   string
	}

	setEnv := func(t *testing.T, in input) {
		t.Helper()

		setOrUnset := func(key, value string) {
			if value != "" {
				t.Setenv(key, value)
			} else {
				t.Setenv(key, "")
				os.Unsetenv(key)
			}
		}

		setOrUnset("HAMCTL_URL", in.url)
		setOrUnset("HAMCTL_OAUTH_IDP_URL", in.oauthIDPURL)
		setOrUnset("HAMCTL_OAUTH_CLIENT_ID", in.oauthClientID)
	}

	buildFlags := func(in input) []string {
		var flags []string
		if in.service != "" {
			flags = append(flags, "--service", in.service)
		}
		if in.httpBaseURL != "" {
			flags = append(flags, "--http-base-url", in.httpBaseURL)
		}
		return flags
	}

	validInput := func() input {
		return input{
			service:       "value",
			httpBaseURL:   "http://localhost",
			oauthIDPURL:   "https://idp.example.com",
			oauthClientID: "client-id",
		}
	}

	tt := []struct {
		name    string
		input   input
		wantErr string // expected error message substring (if any)
	}{
		{
			name:  "success: all inputs provided",
			input: validInput(),
		},
		{
			name: "success: missing http-base-url flag but HAMCTL_URL env var set",
			input: func() input {
				in := validInput()
				in.httpBaseURL = ""
				in.url = "http://localhost"
				return in
			}(),
		},
		{
			name:    "fail: missing HAMCTL_OAUTH_IDP_URL env var",
			input:   func() input { in := validInput(); in.oauthIDPURL = ""; return in }(),
			wantErr: `HAMCTL_OAUTH_IDP_URL`,
		},
		{
			name:    "fail: missing HAMCTL_OAUTH_CLIENT_ID env var",
			input:   func() input { in := validInput(); in.oauthClientID = ""; return in }(),
			wantErr: `HAMCTL_OAUTH_CLIENT_ID`,
		},
		{
			name:    "fail: missing service flag",
			input:   func() input { in := validInput(); in.service = ""; return in }(),
			wantErr: `"service"`,
		},
		{
			name:    "fail: missing http-base-url flag and HAMCTL_URL env var",
			input:   func() input { in := validInput(); in.httpBaseURL = ""; return in }(),
			wantErr: `http-base-url`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// GIVEN an environment configured as we want
			setEnv(t, tc.input)

			// WHEN creating and running the root command
			irrelevantVersion := ""
			cmd, err := NewRoot(&irrelevantVersion)
			if err == nil {
				err = cmd.PersistentFlags().Parse(buildFlags(tc.input))
			}
			if err == nil {
				err = cmd.PersistentPreRunE(cmd, nil)
			}

			// THEN the expected error (or none) is returned
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}
