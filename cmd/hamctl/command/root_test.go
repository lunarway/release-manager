package command

import (
	"fmt"
	"testing"

	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/http"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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

func TestSetCallerEmailFromCommitter(t *testing.T) {
	tt := []struct {
		name             string
		gitConfigApiMock GitConfigAPI
		expectedEmail    string
	}{
		{
			name: "with valid email",
			gitConfigApiMock: &GitConfigAPIMock{
				CommitterDetailsFunc: func() (*git.CommitterDetails, error) {
					return &git.CommitterDetails{Email: "some@email"}, nil
				},
			},
			expectedEmail: "some@email",
		},
		{
			name: "with empty email",
			gitConfigApiMock: &GitConfigAPIMock{
				CommitterDetailsFunc: func() (*git.CommitterDetails, error) {
					return nil, errors.New("could not find email")
				},
			},
			expectedEmail: "",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{}
			setCallerEmailFromCommitter(tc.gitConfigApiMock, client)

			assert.Equal(t, tc.expectedEmail, client.Metadata.CallerEmail)
		})
	}
}

func TestSetCallerEmailFromCommitterReturnsError(t *testing.T) {
	tt := []struct {
		name             string
		gitConfigApiMock GitConfigAPI
		expectedError    error
	}{
		{
			name: "with no email",
			gitConfigApiMock: &GitConfigAPIMock{
				CommitterDetailsFunc: func() (*git.CommitterDetails, error) {
					return nil, errors.New("could not find email")
				},
			},
			expectedError: fmt.Errorf("could not get committer from git: %w", errors.New("could not find email")),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{}
			err := setCallerEmailFromCommitter(tc.gitConfigApiMock, client)
			assert.ErrorContains(t, err, tc.expectedError.Error())
			assert.Equal(t, "", client.Metadata.CallerEmail)
		})
	}
}
