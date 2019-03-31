package http

import (
	"net/http"
	"net/http/httptest"
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
					"builds/product/master/artifact.json",
					"builds/product/master/dev/40-deployment.yaml",
					"builds/product/master/prod/40-deployment.yaml",
					"builds/product/master/staging/40-deployment.yaml",
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
					"builds/product/feature/something-new/artifact.json",
					"builds/product/feature/something-new/dev/40-deployment.yaml",
					"builds/product/feature/something-new/prod/40-deployment.yaml",
					"builds/product/feature/something-new/staging/40-deployment.yaml",
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
					"builds/product/feature/something-new/dev/40-deployment.yaml",
					"builds/product/feature/something-new/prod/40-deployment.yaml",
					"builds/product/feature/something-new/staging/40-deployment.yaml",
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
			branchName, ok := branchName(tc.input.files, tc.input.artifactFileName, tc.input.service)
			assert.Equal(t, tc.ok, ok, "ok bool not as expected")
			assert.Equal(t, tc.branchName, branchName, "name not as expected")
		})
	}
}

func TestAuthenticate(t *testing.T) {
	tt := []struct {
		name          string
		serverToken   string
		authorization string
		status        int
	}{
		{
			name:          "empty authorization",
			serverToken:   "token",
			authorization: "",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "whitespace token",
			serverToken:   "token",
			authorization: "  ",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "non-bearer authorization",
			serverToken:   "token",
			authorization: "non-bearer-token",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "empty bearer authorization",
			serverToken:   "token",
			authorization: "Bearer ",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "whitespace bearer authorization",
			serverToken:   "token",
			authorization: "Bearer      ",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "wrong bearer authorization",
			serverToken:   "token",
			authorization: "Bearer another-token",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "correct bearer authorization",
			serverToken:   "token",
			authorization: "Bearer token",
			status:        http.StatusOK,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tc.authorization)
			w := httptest.NewRecorder()
			authenticate(tc.serverToken, handler)(w, req)

			assert.Equal(t, tc.status, w.Result().StatusCode, "status code not as expected")
		})
	}
}
