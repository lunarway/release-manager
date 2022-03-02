package command

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ pflag.SliceValue = &grafanaOptions{}
var _ pflag.Value = &grafanaOptions{}

func TestGrafanaOptions_String(t *testing.T) {
	tt := []struct {
		name   string
		input  grafanaOptions
		output string
	}{
		{
			name:   "empty",
			input:  grafanaOptions{},
			output: "[]",
		},
		{
			name: "single entry",
			input: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost",
					APIKey: "key",
				},
			},
			output: "dev=<redacted>=localhost",
		},
		{
			name: "multiple entries",
			input: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost1",
					APIKey: "key",
				},
				"prod": grafanaConfig{
					URL:    "localhost2",
					APIKey: "key",
				},
			},
			output: "dev=<redacted>=localhost1,prod=<redacted>=localhost2",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.input.String()

			assert.Equal(t, tc.output, output)
		})
	}
}

func TestGrafanaOptions_Type(t *testing.T) {
	options := grafanaOptions{}

	assert.Equal(t, "<env>=<api-key>=<url>", options.Type())
}

func TestGrafanaOptions_Set(t *testing.T) {
	tt := []struct {
		name   string
		input  string
		output grafanaOptions
		err    error
	}{
		{
			name:   "empty",
			input:  "",
			output: grafanaOptions{},
			err:    errors.New("flag value '': value must be formatted as <env>=<api-key>=<url>"),
		},
		{
			name:  "single complete",
			input: "dev=key=localhost",
			output: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost",
					APIKey: "key",
				},
			},
			err: nil,
		},
		{
			name:  "multiple complete",
			input: "dev=key1=localhost,prod=key2=anotherhost",
			output: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost",
					APIKey: "key1",
				},
				"prod": grafanaConfig{
					URL:    "anotherhost",
					APIKey: "key2",
				},
			},
			err: nil,
		},
		{
			name:  "url with =",
			input: "dev=key1=localhost?query=param",
			output: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost?query=param",
					APIKey: "key1",
				},
			},
			err: nil,
		},
		{
			name:   "multiple values with one incomplete",
			input:  "dev=key1=localhost,prod=key2",
			output: grafanaOptions{},
			err:    errors.New("flag value 'prod=key2': value must be formatted as <env>=<api-key>=<url>"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			opts := grafanaOptions{}

			err := opts.Set(tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.output, opts)
		})
	}
}

func TestGrafanaOptions_GetSlice(t *testing.T) {
	testCases := []struct {
		desc   string
		input  grafanaOptions
		output []string
	}{
		{
			desc:   "empty",
			input:  grafanaOptions{},
			output: nil,
		},
		{
			desc: "single entry",
			input: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost",
					APIKey: "a-key",
				},
			},
			output: []string{
				"dev=<redacted>=localhost",
			},
		},
		{
			desc: "multiple entries",
			input: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost1",
					APIKey: "a-key",
				},
				"prod": grafanaConfig{
					URL:    "localhost2",
					APIKey: "another-key",
				},
			},
			output: []string{
				"dev=<redacted>=localhost1",
				"prod=<redacted>=localhost2",
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			output := tC.input.GetSlice()
			assert.Equal(t, tC.output, output)
		})
	}
}

func TestGrafanaOptions_Append(t *testing.T) {
	tt := []struct {
		name    string
		options grafanaOptions
		input   string
		output  grafanaOptions
		err     error
	}{
		{
			name: "added new entry",
			options: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost",
					APIKey: "key",
				},
			},
			input: "prod=key=localhost2",
			output: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost",
					APIKey: "key",
				},
				"prod": grafanaConfig{
					URL:    "localhost2",
					APIKey: "key",
				},
			},
		},
		{
			name: "added existing entry",
			options: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost",
					APIKey: "key",
				},
			},
			input: "dev=key=localhost2",
			output: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost2",
					APIKey: "key",
				},
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.options.Append(tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.output, tc.options)
		})
	}
}

func TestGrafanaOptions_Replace(t *testing.T) {
	tt := []struct {
		name    string
		options grafanaOptions
		input   []string
		output  grafanaOptions
		err     error
	}{
		{
			name: "single entry",
			options: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost1",
					APIKey: "key",
				},
			},
			input: []string{"prod=key=localhost2"},
			output: grafanaOptions{
				"prod": grafanaConfig{
					URL:    "localhost2",
					APIKey: "key",
				},
			},
		},
		{
			name: "equal entry with addition",
			options: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost1",
					APIKey: "key",
				},
			},
			input: []string{"dev=key=localhost", "prod=key=localhost2"},
			output: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost",
					APIKey: "key",
				},
				"prod": grafanaConfig{
					URL:    "localhost2",
					APIKey: "key",
				},
			},
		},
		{
			name: "bad entry",
			options: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost1",
					APIKey: "key",
				},
			},
			input: []string{"dev=key"},
			output: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost",
					APIKey: "key",
				},
			},
			err: errors.New("value must be formatted as <env>=<api-key>=<url>"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.options.Replace(tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.output, tc.options)
		})
	}
}
