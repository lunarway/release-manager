package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchName(t *testing.T) {
	type input struct {
		files            []string
		artifactFileName string
		service          string
	}
	tt := []struct {
		name       string
		input      input
		branchName string
		ok         bool
	}{
		{
			name: "nil slice",
			input: input{
				files:            nil,
				artifactFileName: "artifact.json",
				service:          "test",
			},
			branchName: "",
			ok:         false,
		},
		{
			name: "empty files slice",
			input: input{
				files:            []string{},
				artifactFileName: "artifact.json",
				service:          "test",
			},
			branchName: "",
			ok:         false,
		},
		{
			name: "files from a build from master",
			input: input{
				files: []string{
					"artifacts/product/master/artifact.json",
					"artifacts/product/master/dev/40-deployment.yaml",
					"artifacts/product/master/prod/40-deployment.yaml",
					"artifacts/product/master/staging/40-deployment.yaml",
				},
				artifactFileName: "artifact.json",
				service:          "product",
			},
			branchName: "master",
			ok:         true,
		},
		{
			name: "files from a build on a branch with slashes",
			input: input{
				files: []string{
					"artifacts/product/feature/something-new/artifact.json",
					"artifacts/product/feature/something-new/dev/40-deployment.yaml",
					"artifacts/product/feature/something-new/prod/40-deployment.yaml",
					"artifacts/product/feature/something-new/staging/40-deployment.yaml",
				},
				artifactFileName: "artifact.json",
				service:          "product",
			},
			branchName: "feature/something-new",
			ok:         true,
		},
		{
			name: "files from a build on master but no artifact.json",
			input: input{
				files: []string{
					"artifacts/product/feature/something-new/dev/40-deployment.yaml",
					"artifacts/product/feature/something-new/prod/40-deployment.yaml",
					"artifacts/product/feature/something-new/staging/40-deployment.yaml",
				},
				artifactFileName: "artifact.json",
				service:          "product",
			},
			branchName: "",
			ok:         false,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			branchName, ok := BranchName(tc.input.files, tc.input.artifactFileName, tc.input.service)
			assert.Equal(t, tc.ok, ok, "ok bool not as expected")
			assert.Equal(t, tc.branchName, branchName, "name not as expected")
		})
	}
}
