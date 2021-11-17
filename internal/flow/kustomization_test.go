package flow

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKustomizationExists(t *testing.T) {
	tt := []struct {
		name    string
		testDir string
		path    string
	}{
		{
			name:    "toolkit v1beta1",
			testDir: "flux-toolkit-kustomization-v1beta1",
			path:    "testdata/flux-toolkit-kustomization-v1beta1/flux-toolkit-kustomization-v1beta1.yaml",
		},
		{
			name:    "toolkit v1",
			testDir: "flux-toolkit-kustomization-v1",
			path:    "testdata/flux-toolkit-kustomization-v1/flux-toolkit-kustomization-v1.yaml",
		},
		{
			name:    "config",
			testDir: "config-kustomization",
			path:    "",
		},
		{
			name:    "kustomization not found",
			testDir: "no-kustomization",
			path:    "",
		},
		{
			name:    "empty-file",
			testDir: "empty-file",
			path:    "",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := kustomizationExists(path.Join("testdata", tc.testDir))
			require.NoError(t, err)

			assert.Equal(t, tc.path, actual)
		})
	}
}
