package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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
			name: "multiple whitespace strings",
			input: strings("  ", "	"),
			output: nil,
		},
		{
			name: "mixed whitespace and non-whitespace strings",
			input: strings("  ", "hello", "	", "world"),
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

func TestPolicyPatchPath(t *testing.T) {
	tt := []struct {
		name       string
		path       string
		valid      bool
		policyType string
	}{
		{
			name:       "empty",
			path:       "http://localhost",
			valid:      false,
			policyType: "",
		},
		{
			name:       "no type",
			path:       "http://localhost/policies",
			valid:      false,
			policyType: "",
		},
		{
			name:       "with type",
			path:       "http://localhost/policies/auto-release",
			valid:      true,
			policyType: "auto-release",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPatch, tc.path, nil)
			p, ok := newPolicyPatchPath(r)
			assert.Equal(t, tc.valid, ok, "path not valid")
			if !ok {
				return
			}
			policyType := p.PolicyType()
			assert.Equal(t, tc.policyType, policyType, "policy type not as expected")
		})
	}
}
