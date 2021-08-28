package command

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrafanaOptions(t *testing.T) {
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
			input: "dev=key1=localhost,staging=key2=anotherhost",
			output: grafanaOptions{
				"dev": grafanaConfig{
					URL:    "localhost",
					APIKey: "key1",
				},
				"staging": grafanaConfig{
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
			input:  "dev=key1=localhost,staging=key2",
			output: grafanaOptions{},
			err:    errors.New("flag value 'staging=key2': value must be formatted as <env>=<api-key>=<url>"),
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
