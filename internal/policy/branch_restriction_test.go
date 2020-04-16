package policy

import (
	"context"
	"testing"

	"github.com/lunarway/release-manager/internal/copy"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zapcore"
	git "gopkg.in/src-d/go-git.v4"
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
				{
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
				{
					Environment: "prod",
					BranchRegex: "master",
				},
				{
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
				{
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
				{
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
				{
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
				{
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

func TestService_ApplyBranchRestriction(t *testing.T) {
	tt := []struct {
		name        string
		svc         string
		branchRegex string
		env         string

		globalPolicies []BranchRestriction

		id      string
		polcies Policies
		err     error
	}{
		{
			name:        "invaid branch regex",
			svc:         "empty",
			branchRegex: "^master(",
			env:         "prod",
			id:          "",
			err:         errors.New("branch regex not valid: error parsing regexp: missing closing ): `^master(`"),
		},
		{
			name:        "conflict with auto release",
			svc:         "autorelease",
			branchRegex: "^dev$",
			env:         "dev",
			id:          "",
			polcies: Policies{
				Service: "autorelease",
				AutoReleases: []AutoReleasePolicy{
					{
						ID:          "auto-release-master-dev",
						Environment: "dev",
						Branch:      "master",
					},
				},
			},
			err: errors.New("conflict with auto-release-master-dev: conflict"),
		},
		{
			name:        "match with auto release",
			svc:         "autorelease",
			branchRegex: "^master$",
			env:         "dev",
			id:          "branch-restriction-dev",
			polcies: Policies{
				Service: "autorelease",
				AutoReleases: []AutoReleasePolicy{
					{
						ID:          "auto-release-master-dev",
						Environment: "dev",
						Branch:      "master",
					},
				},
				BranchRestrictions: []BranchRestriction{
					{
						ID:          "branch-restriction-dev",
						Environment: "dev",
						BranchRegex: "^master$",
					},
				},
			},
			err: nil,
		},
		{
			name:        "conflict with global restriction",
			svc:         "empty",
			branchRegex: "^dev$",
			env:         "dev",
			globalPolicies: []BranchRestriction{
				{
					Environment: "dev",
					BranchRegex: "^master$",
				},
			},
			id: "",
			polcies: Policies{
				Service: "empty",
				BranchRestrictions: []BranchRestriction{
					{
						ID:          "",
						Environment: "dev",
						BranchRegex: "^master$",
					},
				},
			},
			err: errors.New("conflicts with global policy: conflict"),
		},
		{
			name:        "valid new restriction",
			svc:         "empty",
			branchRegex: "^master$",
			env:         "prod",
			globalPolicies: []BranchRestriction{
				{
					Environment: "dev",
					BranchRegex: "^master$",
				},
			},
			id: "branch-restriction-prod",
			polcies: Policies{
				Service: "empty",
				BranchRestrictions: []BranchRestriction{
					{
						ID:          "",
						Environment: "dev",
						BranchRegex: "^master$",
					},
					{
						ID:          "branch-restriction-prod",
						Environment: "prod",
						BranchRegex: "^master$",
					},
				},
			},
			err: nil,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			log.Init(&log.Configuration{
				Level: log.Level{
					Level: zapcore.DebugLevel,
				},
				Development: true,
			})
			gitService := MockGitService{}
			var destinationPath string
			gitService.On("MasterPath").Return(func() string {
				if destinationPath != "" {
					return destinationPath
				}
				return "testdata"
			})
			gitService.On("Clone", mock.Anything, mock.Anything).Return(func(ctx context.Context, path string) *git.Repository {
				// store the destination path. It will be a temporary directory so we
				// need it to later inspect the results of the operation
				destinationPath = path
				// copy testdata into path faking a master clone
				err := copy.CopyDir(ctx, "testdata", path)
				assert.NoError(t, err, "unexpected error when copying in Clone")
				return nil
			}, nil)
			gitService.On("Commit", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			s := Service{
				Tracer:                          tracing.NewNoop(),
				Git:                             &gitService,
				GlobalBranchRestrictionPolicies: tc.globalPolicies,
			}

			id, err := s.ApplyBranchRestriction(context.Background(), Actor{
				Email: "test@lunar.app",
				Name:  "Test",
			}, tc.svc, tc.branchRegex, tc.env)

			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error(), "error not as expected")
			} else {
				assert.NoError(t, err, "unexpected error")
			}
			assert.Equal(t, tc.id, id, "id not as expected")

			// read the stored policies from the destination path
			policies, err := s.Get(context.Background(), tc.svc)
			if err != nil {
				t.Logf("Get stored policy failed: %v", err)
			}
			assert.Equal(t, tc.polcies, policies, "updated policies not as expected")
		})
	}
}
