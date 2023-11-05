package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
			name:          "empty bearer authorization",
			serverToken:   "token",
			authorization: " ",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "whitespace bearer authorization",
			serverToken:   "token",
			authorization: "      ",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "wrong bearer authorization",
			serverToken:   "token",
			authorization: "another-token",
			status:        http.StatusUnauthorized,
		},
		{
			name:          "correct bearer authorization",
			serverToken:   "token",
			authorization: "token",
			status:        http.StatusOK,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("X-HAM-TOKEN", tc.authorization)
			w := httptest.NewRecorder()
			authenticate(tc.serverToken)(handler).ServeHTTP(w, req)

			assert.Equal(t, tc.status, w.Result().StatusCode, "status code not as expected")
		})
	}
}
