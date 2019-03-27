package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlattenHeaders(t *testing.T) {
	tt := []struct {
		name   string
		input  map[string][]string
		output map[string]string
	}{
		{
			name: "single value header",
			input: map[string][]string{
				"User-Agent": []string{"curl"},
			},
			output: map[string]string{
				"User-Agent": "curl",
			},
		},
		{
			name: "multi value header",
			input: map[string][]string{
				"User-Agent": []string{"curl", "1.2.3"},
			},
			output: map[string]string{
				"User-Agent": "curl,1.2.3",
			},
		},
		{
			name: "multiple mixed headers",
			input: map[string][]string{
				"Authoriztaion": []string{"Bearer token"},
				"User-Agent":    []string{"curl", "1.2.3"},
			},
			output: map[string]string{
				"Authoriztaion": "Bearer token",
				"User-Agent":    "curl,1.2.3",
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			output := flattenHeaders(tc.input)
			assert.Equal(t, tc.output, output, "headers not as expected")
		})
	}
}

func TestSecureHeaders(t *testing.T) {
	tt := []struct {
		name   string
		input  map[string]string
		output map[string]string
	}{
		{
			name: "no Authorization",
			input: map[string]string{
				"User-Agent": "curl",
			},
			output: map[string]string{
				"User-Agent": "curl",
			},
		},
		{
			name: "non-bearer short Authorization",
			input: map[string]string{
				"Authorization": "curl",
			},
			output: map[string]string{
				"Authorization": "curl",
			},
		},
		{
			name: "non-Bearer long Authorization",
			input: map[string]string{
				"Authorization": "curl12345678",
			},
			output: map[string]string{
				"Authorization": "curl",
			},
		},
		{
			name: "empty Bearer Authorization",
			input: map[string]string{
				"Authorization": "Bearer",
			},
			output: map[string]string{
				"Authorization": "Bearer",
			},
		},
		{
			name: "single character Bearer Authorization",
			input: map[string]string{
				"Authorization": "Bearer 1",
			},
			output: map[string]string{
				"Authorization": "Bearer 1",
			},
		},
		{
			name: "four character Bearer Authorization",
			input: map[string]string{
				"Authorization": "Bearer 1234",
			},
			output: map[string]string{
				"Authorization": "Bearer 1234",
			},
		},
		{
			name: "five character Bearer Authorization",
			input: map[string]string{
				"Authorization": "Bearer 12345",
			},
			output: map[string]string{
				"Authorization": "Bearer 1234",
			},
		},
		{
			name: "long Bearer Authorization",
			input: map[string]string{
				"Authorization": "Bearer 12345678901234567890",
			},
			output: map[string]string{
				"Authorization": "Bearer 1234",
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			output := secureHeaders(tc.input)
			assert.Equal(t, tc.output, output, "headers not as expected")
		})
	}
}
