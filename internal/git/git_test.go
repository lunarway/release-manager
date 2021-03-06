package git

import (
	"testing"

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
