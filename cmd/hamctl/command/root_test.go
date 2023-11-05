package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultShuttleString(t *testing.T) {
	tt := []struct {
		name      string
		flagValue string
		spec      shuttleSpec
		output    string
	}{
		{
			name:      "flag value",
			flagValue: "value",
			spec: shuttleSpec{
				Vars: shuttleSpecVars{
					Service: "service",
				},
			},
			output: "value",
		},
		{
			name:      "empty flag value",
			flagValue: "",
			spec: shuttleSpec{
				Vars: shuttleSpecVars{
					Service: "service",
				},
			},
			output: "service",
		},
		{
			name:      "whitespace flag value",
			flagValue: "   ",
			spec: shuttleSpec{
				Vars: shuttleSpecVars{
					Service: "service",
				},
			},
			output: "service",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			defaultShuttleString(func() (shuttleSpec, bool) {
				return tc.spec, true
			}, &tc.flagValue, func(s *shuttleSpec) string {
				return s.Vars.Service
			})
			assert.Equal(t, tc.output, tc.flagValue, "flag value not as expected")
		})
	}
}

func TestDefaultShuttleString_noSpec(t *testing.T) {
	var flagValue string
	defaultShuttleString(func() (shuttleSpec, bool) {
		return shuttleSpec{}, false
	}, &flagValue, func(s *shuttleSpec) string {
		return s.Vars.Service
	})
	assert.Equal(t, "", flagValue, "flag value not as expected")
}
