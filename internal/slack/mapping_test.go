package slack

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUserMappings(t *testing.T) {
	tt := []struct {
		name   string
		input  []string
		output map[string]string
		err    error
	}{
		{
			name:   "nil slice",
			input:  nil,
			output: map[string]string{},
			err:    nil,
		},
		{
			name:   "empty slice",
			input:  []string{},
			output: map[string]string{},
			err:    nil,
		},
		{
			name: "single empty entry",
			input: []string{
				"",
			},
			output: map[string]string{},
			err:    nil,
		},
		{
			name: "single whitespace entry",
			input: []string{
				" ",
			},
			output: map[string]string{},
			err:    nil,
		},
		{
			name: "single email entry",
			input: []string{
				"foo@bar.com=foo@lunarway.com",
			},
			output: map[string]string{
				"foo@bar.com": "foo@lunarway.com",
			},
			err: nil,
		},
		{
			name: "multiple different email entries",
			input: []string{
				"foo@bar.com=foo@lunarway.com",
				"bar@foo.com=bar@lunarway.com",
			},
			output: map[string]string{
				"foo@bar.com": "foo@lunarway.com",
				"bar@foo.com": "bar@lunarway.com",
			},
			err: nil,
		},
		{
			name: "single mapping with whitespace",
			input: []string{
				"foo@bar.com=  foo@lunarway.com",
			},
			output: map[string]string{
				"foo@bar.com": "foo@lunarway.com",
			},
			err: nil,
		},
		{
			name: "multiple conflicting email entries",
			input: []string{
				"foo@bar.com=foo@lunarway.com",
				"foo@bar.com=bar@lunarway.com",
			},
			output: nil,
			err:    errors.New("conflicting user mappings for foo@bar.com"),
		},
		{
			name: "incomplete mapping of source",
			input: []string{
				"=foo@bar.com",
			},
			output: nil,
			err:    errors.New("invalid user mapping '=foo@bar.com'"),
		},
		{
			name: "incomplete mapping of destination",
			input: []string{
				"foo@bar.com=",
			},
			output: nil,
			err:    errors.New("invalid user mapping 'foo@bar.com='"),
		},
		{
			name: "missing equal sign",
			input: []string{
				"foo@bar.com",
			},
			output: nil,
			err:    errors.New("invalid user mapping 'foo@bar.com'"),
		},
		{
			name: "multiple equal signs",
			input: []string{
				"foo@bar.com=foo@lunarway.com=bar@lunarway.com",
			},
			output: nil,
			err:    errors.New("invalid user mapping 'foo@bar.com=foo@lunarway.com=bar@lunarway.com'"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			output, err := ParseUserMappings(tc.input)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error(), "output error not as expected")
			} else {
				assert.NoError(t, err, "unexpected output error")
			}
			assert.Equal(t, tc.output, output, "output not as expected")
		})
	}
}
