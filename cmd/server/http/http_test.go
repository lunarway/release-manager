package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

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

func TestCommitMessageExtraction(t *testing.T) {
	tt := []struct {
		name          string
		commitMessage string
		commitInfo    commitInfo
		err           error
	}{
		{
			name:          "exact values",
			commitMessage: "[test-service] artifact master-1234ds13g3-12s46g356g by Foo Bar\nSigned-off-by: Foo Bar <test@lunar.app>",
			commitInfo: commitInfo{
				AuthorEmail: "test@lunar.app",
				AuthorName:  "Foo Bar",
				Service:     "test-service",
			},
			err: nil,
		},
		{
			name:          "missing signoff",
			commitMessage: "[product] build something",
			commitInfo:    commitInfo{},
			err:           errors.New("not enough matches"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			info, err := extractAuthorFromCommit(tc.commitMessage)
			if tc.err != nil {
				assert.EqualError(t, errors.Cause(err), tc.err.Error(), "output error not as expected")
			} else {
				assert.NoError(t, err, "no output error expected")
			}
			assert.Equal(t, tc.commitInfo.AuthorName, info.AuthorName, "AuthorName not as expected")
			assert.Equal(t, tc.commitInfo.Service, info.Service, "Service not as expected")
			assert.Equal(t, tc.commitInfo.AuthorEmail, info.AuthorEmail, "AuthorEmail not as expected")
		})
	}
}
