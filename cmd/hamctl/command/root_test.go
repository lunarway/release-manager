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

func TestSetCallerEmail(t *testing.T) {
	tt := []struct {
		name             string
		gitConfigApiMock GitConfigAPI
		email            string
		expectedEmail    string
	}{
		{
			name: "with valid email from committer",
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
		{
			name:          "with valid email",
			email:         "some@email",
			expectedEmail: "some@email",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{}
			_ = setCallerEmail(tc.gitConfigApiMock, client, tc.email)
			assert.Equal(t, tc.expectedEmail, client.Metadata.CallerEmail)
		})
	}
}

func TestSetCallerEmailReturnsError(t *testing.T) {
	tt := []struct {
		name             string
		gitConfigApiMock GitConfigAPI
		email            string
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
			err := setCallerEmail(tc.gitConfigApiMock, client, tc.email)
			assert.ErrorContains(t, err, tc.expectedError.Error())
			assert.Equal(t, "", client.Metadata.CallerEmail)
		})
	}
}
