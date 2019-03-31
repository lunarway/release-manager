package policy

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tt := []struct {
		name     string
		input    io.Reader
		policies Policies
		err      error
	}{
		{
			name:     "empty reader",
			input:    strings.NewReader(""),
			policies: Policies{},
			err:      nil,
		},
		{
			name:     "non-json contents",
			input:    strings.NewReader("hello world"),
			policies: Policies{},
			err:      ErrNotParsable,
		},
		{
			name:     "unknown json fields",
			input:    strings.NewReader(`{"unknown": "field"}`),
			policies: Policies{},
			err:      ErrUnknownFields,
		},
		{
			name:  "valid policies",
			input: strings.NewReader(`{"service": "test", "autoReleases": [{ "id": "master-dev" }]}`),
			policies: Policies{
				Service: "test",
				AutoReleases: []AutoReleasePolicy{
					{
						ID: "master-dev",
					},
				},
			},
			err: nil,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			policies, err := parse(tc.input)
			if tc.err != nil {
				assert.EqualError(t, errors.Cause(err), tc.err.Error(), "output error not as expected")
			} else {
				assert.NoError(t, err, "no output error expected")
			}
			assert.Equal(t, tc.policies, policies, "policies not as expected")
		})
	}
}

func TestPolicies_SetAutoRelease(t *testing.T) {
	// helper func to create auto release
	// variadic arguments are parsed in {branch,env} pairs
	policy := func(args ...string) Policies {
		if len(args)%2 != 0 {
			t.Fatalf("uneven number of arguments to policy creator")
		}
		p := Policies{}
		for i := 0; i < len(args); i = i + 2 {
			branch := args[i]
			env := args[i+1]
			p.AutoReleases = append(p.AutoReleases, AutoReleasePolicy{
				ID:          fmt.Sprintf("auto-release-%s-%s", branch, env),
				Branch:      branch,
				Environment: env,
			})
		}
		return p
	}
	type input struct {
		policies Policies
		branch   string
		env      string
	}
	type output struct {
		policies Policies
		id       string
	}
	tt := []struct {
		name   string
		input  input
		output output
	}{
		{
			name: "empty policies",
			input: input{
				policies: Policies{},
				branch:   "master",
				env:      "dev",
			},
			output: output{
				policies: policy("master", "dev"),
				id:       "auto-release-master-dev",
			},
		},
		{
			name: "existing policy for same env and branch",
			input: input{
				policies: policy("master", "dev"),
				branch:   "master",
				env:      "dev",
			},
			output: output{
				policies: policy("master", "dev"),
				id:       "auto-release-master-dev",
			},
		},
		{
			name: "existing policy on env and another branch",
			input: input{
				policies: policy("master", "dev"),
				branch:   "feature",
				env:      "dev",
			},
			output: output{
				policies: policy("feature", "dev"),
				id:       "auto-release-feature-dev",
			},
		},
		{
			name: "existing policy for another env and branch",
			input: input{
				policies: policy("master", "dev"),
				branch:   "feature",
				env:      "staging",
			},
			output: output{
				policies: policy(
					"master", "dev",
					"feature", "staging",
				),
				id: "auto-release-feature-staging",
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("input: %v", tc.input.policies.AutoReleases)
			id := tc.input.policies.SetAutoRelease(tc.input.branch, tc.input.env)
			t.Logf("output: %v", tc.input.policies.AutoReleases)
			assert.Equal(t, tc.output.id, id, "policy ID not as expected")
			assert.Equal(t, tc.output.policies, tc.input.policies, "policies not as expected")
		})
	}
}

func TestPolicies_Delete(t *testing.T) {
	policy := func(ids ...string) Policies {
		p := Policies{}
		for _, id := range ids {
			p.AutoReleases = append(p.AutoReleases, AutoReleasePolicy{
				ID: id,
			})
		}
		return p
	}
	ids := func(ids ...string) []string {
		return ids
	}
	type input struct {
		policies Policies
		ids      []string
	}
	type output struct {
		policies Policies
		count    int
	}
	tt := []struct {
		name   string
		input  input
		output output
	}{
		{
			name: "empty policies no ids",
			input: input{
				policies: policy(),
				ids:      ids(),
			},
			output: output{
				policies: policy(),
				count:    0,
			},
		},
		{
			name: "empty policies nil ids",
			input: input{
				policies: policy(),
				ids:      nil,
			},
			output: output{
				policies: policy(),
				count:    0,
			},
		},
		{
			name: "no matching ids",
			input: input{
				policies: policy("id-1", "id-2"),
				ids:      ids("id-3", "id-4"),
			},
			output: output{
				policies: policy("id-1", "id-2"),
				count:    0,
			},
		},
		{
			name: "single matching id",
			input: input{
				policies: policy("id-1", "id-2"),
				ids:      ids("id-1"),
			},
			output: output{
				policies: policy("id-2"),
				count:    1,
			},
		},
		{
			name: "all matching ids",
			input: input{
				policies: policy("id-1", "id-2"),
				ids:      ids("id-2", "id-1"),
			},
			output: output{
				policies: policy(),
				count:    2,
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("input: %v", tc.input.policies.AutoReleases)
			count := tc.input.policies.Delete(tc.input.ids...)
			t.Logf("output: %v", tc.input.policies.AutoReleases)
			assert.Equal(t, tc.output.count, count, "deleted count not as expected")
			assert.Equal(t, tc.output.policies, tc.input.policies, "policies not as expected")
		})
	}
}
