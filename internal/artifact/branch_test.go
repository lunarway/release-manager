package artifact_test

import (
	"testing"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/stretchr/testify/assert"
)

const (
	branchWithoutSlash = "master"
	branchWithSlash    = "krvi/foo"
	normalizedBranch   = "krvi_foo"
)

func TestNormalizeBranch(t *testing.T) {
	tt := []struct {
		name   string
		branch string
		output string
	}{
		{
			name:   "empty",
			branch: "",
			output: "",
		},
		{
			name:   "without slash",
			branch: branchWithoutSlash,
			output: branchWithoutSlash,
		},
		{
			name:   "already normalized",
			branch: normalizedBranch,
			output: normalizedBranch,
		},
		{
			name:   "single slash",
			branch: branchWithSlash,
			output: normalizedBranch,
		},
		{
			name:   "multiple slashes",
			branch: "krvi/foo/bar",
			output: "krvi_foo_bar",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.output, artifact.NormalizeBranch(tc.branch))
		})
	}
}

func TestSameBranch(t *testing.T) {
	tt := []struct {
		name string
		a    string
		b    string
		same bool
	}{
		{
			name: "identical without slash",
			a:    branchWithoutSlash,
			b:    branchWithoutSlash,
			same: true,
		},
		{
			name: "identical with slash",
			a:    branchWithSlash,
			b:    branchWithSlash,
			same: true,
		},
		{
			name: "raw and normalized",
			a:    branchWithSlash,
			b:    normalizedBranch,
			same: true,
		},
		{
			name: "normalized and raw",
			a:    normalizedBranch,
			b:    branchWithSlash,
			same: true,
		},
		{
			name: "different branches",
			a:    branchWithSlash,
			b:    "krvi/bar",
			same: false,
		},
		{
			name: "other separator",
			a:    branchWithSlash,
			b:    "krvi-foo",
			same: false,
		},
		{
			name: "prefix",
			a:    branchWithSlash,
			b:    "krvi/foo-bar",
			same: false,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.same, artifact.SameBranch(tc.a, tc.b))
		})
	}
}
