package git

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCredentials(t *testing.T) {
	tt := []struct {
		name     string
		paths    []string
		userName string
		email    string
		err      error
	}{
		{
			name: "complete set",
			paths: []string{
				"testdata/user_set_1",
			},
			userName: "Foo",
			email:    "foo@foo.com",
			err:      nil,
		},
		{
			name: "first path missing email",
			paths: []string{
				"testdata/email_missing",
				"testdata/user_set_1",
			},
			userName: "Foo",
			email:    "foo@foo.com",
			err:      nil,
		},
		{
			name: "first path missing name",
			paths: []string{
				"testdata/name_missing",
				"testdata/user_set_1",
			},
			userName: "Foo",
			email:    "foo@foo.com",
			err:      nil,
		},
		{
			name: "configuration file not found in first path",
			paths: []string{
				"testdata/unknown_path",
				"testdata/user_set_1",
			},
			userName: "Foo",
			email:    "foo@foo.com",
			err:      nil,
		},
		{
			name: "configuration file not found in all paths",
			paths: []string{
				"testdata/unknown_path_1",
				"testdata/unknown_path_2",
			},
			userName: "",
			email:    "",
			err:      errors.New("failed to read Git credentials from paths: [testdata/unknown_path_1 testdata/unknown_path_2]"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			userName, email, err := credentials(tc.paths...)
			t.Logf("error: %v", err)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error(), "output error not as expected")
			} else {
				assert.NoError(t, err, "unexpected output error")
			}
			assert.Equal(t, tc.userName, userName, "user name not as expected")
			assert.Equal(t, tc.email, email, "email not as expected")
		})
	}
}

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

func TestLocateArtifactCondition(t *testing.T) {
	tt := []struct {
		name       string
		artifactID string
		message    string
		output     bool
		err        error
	}{
		{
			name:       "empty artifact id",
			artifactID: "",
			message:    "[service-name] artifact master-1234567890-1234567890 by Author",
			output:     false,
		},
		{
			name:       "regexp like artifact id",
			artifactID: `(\`,
			message:    "[service-name] artifact master-1234567890-1234567890 by Author",
			output:     false,
		},
		{
			name:       "partial artifact id",
			artifactID: "master-1234",
			message:    "[service-name] artifact master-1234567890-1234567890 by Author",
			output:     false,
		},
		{
			name:       "partial artifact id with complete application hash",
			artifactID: "master-1234567890",
			message:    "[service-name] artifact master-1234567890-1234567890 by Author",
			output:     false,
		},
		{
			name:       "exact artifact id",
			artifactID: "master-1234567890-1234567890",
			message:    "[service-name] artifact master-1234567890-1234567890 by Author",
			output:     true,
		},
		{
			name:       "wrong cased artifact id",
			artifactID: "MASTER-1234567890-1234567890",
			message:    "[service-name] artifact master-1234567890-1234567890 by Author",
			output:     true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			output := locateArtifactCondition(tc.artifactID)(tc.message)
			assert.Equal(t, tc.output, output, "output not as expected")
		})
	}
}