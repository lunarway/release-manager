package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	policyinternal "github.com/lunarway/release-manager/internal/policy"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestFilterEmptyStrings(t *testing.T) {
	strings := func(s ...string) []string {
		return s
	}
	tt := []struct {
		name   string
		input  []string
		output []string
	}{
		{
			name:   "nil input",
			input:  nil,
			output: nil,
		},
		{
			name:   "empty slice",
			input:  strings(),
			output: nil,
		},
		{
			name:   "single whitespace string",
			input:  strings("  "),
			output: nil,
		},
		{
			name:   "multiple whitespace strings",
			input:  strings("  ", "	"),
			output: nil,
		},
		{
			name:   "mixed whitespace and non-whitespace strings",
			input:  strings("  ", "hello", "	", "world"),
			output: strings("hello", "world"),
		},
		{
			name:   "single non-whitespace string",
			input:  strings("hello"),
			output: strings("hello"),
		},
		{
			name:   "multiple non-whitespace strings",
			input:  strings("hello", "world"),
			output: strings("hello", "world"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			output := filterEmptyStrings(tc.input)
			assert.Equal(t, tc.output, output, "output not as expected")
		})
	}
}

func TestApplyBranchRestrictionPolicy_invalidBranchRegex(t *testing.T) {
	tt := []struct {
		name        string
		branchRegex string
		message     string
	}{
		{
			name:        "not a regular expression",
			branchRegex: "^master(",
			message:     "branch regex not valid: error parsing regexp: missing closing ): `^master(`",
		},
		{
			name:        "contains slash",
			branchRegex: "^krvi/.*$",
			message:     policyinternal.ErrBranchRegexContainsSlash.Error(),
		},
	}
	log.Init(&log.Configuration{
		Level: log.Level{
			Level: zapcore.DebugLevel,
		},
		Development: true,
	})
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(httpinternal.ApplyBranchRestrictionPolicyRequest{
				Service:     "example",
				Environment: "dev",
				BranchRegex: tc.branchRegex,
			})
			require.NoError(t, err)
			path := "/policies/branch-restriction"
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, path, bytes.NewReader(body))
			w := httptest.NewRecorder()
			policySvc := &policyinternal.Service{Tracer: tracing.NewNoop()}
			handler := applyBranchRestrictionPolicy(&payload{tracer: tracing.NewNoop()}, policySvc)

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code, "status code not as expected")
			var resp httpinternal.ErrorResponse
			require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
			assert.Equal(t, tc.message, resp.Message, "error message not as expected")
		})
	}
}
