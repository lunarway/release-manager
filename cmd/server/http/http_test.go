package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
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
			logger := log.New(&log.Configuration{
				Level: log.Level{
					Level: zapcore.DebugLevel,
				},
				Development: true,
			})
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tc.authorization)
			w := httptest.NewRecorder()
			authenticate(logger, tc.serverToken)(handler).ServeHTTP(w, req)

			assert.Equal(t, tc.status, w.Result().StatusCode, "status code not as expected")
		})
	}
}
