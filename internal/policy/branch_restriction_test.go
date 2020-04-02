package policy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanRelease(t *testing.T) {
	tt := []struct {
		name         string
		branch       string
		env          string
		restrictions []BranchRestriction
		canRelease   bool
	}{
		{
			name:         "no policies",
			branch:       "branch",
			env:          "dev",
			restrictions: nil,
			canRelease:   true,
		},
		{
			name:   "single policy environment not matching",
			branch: "branch",
			env:    "dev",
			restrictions: []BranchRestriction{
				BranchRestriction{
					Environment: "prod",
					BranchRegex: "master",
				},
			},
			canRelease: true,
		},
		{
			name:   "multiple policis environment not matching",
			branch: "branch",
			env:    "dev",
			restrictions: []BranchRestriction{
				BranchRestriction{
					Environment: "prod",
					BranchRegex: "master",
				},
				BranchRestriction{
					Environment: "staging",
					BranchRegex: "master",
				},
			},
			canRelease: true,
		},
		{
			name:   "environment restricted to another branch",
			branch: "branch",
			env:    "dev",
			restrictions: []BranchRestriction{
				BranchRestriction{
					Environment: "dev",
					BranchRegex: "master",
				},
			},
			canRelease: false,
		},
		{
			name:   "environment restricted to same branch",
			branch: "master",
			env:    "dev",
			restrictions: []BranchRestriction{
				BranchRestriction{
					Environment: "dev",
					BranchRegex: "master",
				},
			},
			canRelease: true,
		},
		{
			// specifically tests non-limited regular expressions. This is to document
			// that this is intended behaviour and that branch restrictions must be as
			// limited as possible
			name:   "environment restricted to branch with same prefix and loose branch regex",
			branch: "master-update",
			env:    "dev",
			restrictions: []BranchRestriction{
				BranchRestriction{
					Environment: "dev",
					BranchRegex: "master",
				},
			},
			canRelease: true,
		},
		{
			name:   "environment restricted to branch with same prefix and strong branch regex",
			branch: "master-update",
			env:    "dev",
			restrictions: []BranchRestriction{
				BranchRestriction{
					Environment: "dev",
					BranchRegex: "^master$",
				},
			},
			canRelease: false,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			policies := Policies{
				BranchRestrictions: tc.restrictions,
			}
			ok, _ := canRelease(context.Background(), policies, tc.branch, tc.env)
			assert.Equal(t, tc.canRelease, ok, "can release boolean not as expected")
		})
	}
}
