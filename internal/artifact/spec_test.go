package artifact_test

import (
	"path"
	"strings"
	"testing"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	tt := []struct {
		name string
		path string
		spec artifact.Spec
		err  error
	}{
		{
			name: "existing and valid artifact",
			path: "valid_artifact.json",
			spec: artifact.Spec{
				ID: "valid",
			},
		},
		{
			name: "unknown artifact",
			path: "unknown_artifact.json",
			spec: artifact.Spec{},
			err:  artifact.ErrFileNotFound,
		},
		{
			name: "invalid artifact",
			path: "invalid_artifact.json",
			spec: artifact.Spec{},
			err:  artifact.ErrNotParsable,
		},
		{
			name: "unknown fields in artifact",
			path: "unknown_fields_artifact.json",
			spec: artifact.Spec{},
			err:  artifact.ErrUnknownFields,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := artifact.Get(path.Join("testdata", tc.path))
			if tc.err != nil {
				assert.EqualError(t, errors.Cause(err), tc.err.Error(), "output error not as expected")
			} else {
				assert.NoError(t, err, "no output error expected")
			}
			assert.Equal(t, tc.spec, spec, "spec not as expected")
		})
	}
}

func TestDecode(t *testing.T) {
	tt := []struct {
		name   string
		input  string
		output artifact.Spec
	}{
		{
			name:  "no stages",
			input: `{"id": "no-stages"}`,
			output: artifact.Spec{
				ID: "no-stages",
			},
		},
		{
			name: "with stages",
			input: `
			{
				"id": "stages",
				"stages": [
					{
						"id": "build",
						"name": "name",
						"data": {
							"image": "quay.io/lunarway/release-manager",
							"tag": "v1.2.3",
							"dockerVersion": "20.10.6"
						}
					}
				]
			}
`,
			output: artifact.Spec{
				ID: "stages",
				Stages: []artifact.Stage{
					{
						ID:   artifact.StageIDBuild,
						Name: "name",
						Data: artifact.BuildData{
							Image:         "quay.io/lunarway/release-manager",
							Tag:           "v1.2.3",
							DockerVersion: "20.10.6",
						},
					},
				},
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := artifact.Decode(strings.NewReader(tc.input))

			require.NoError(t, err, "unexpected error")
			require.Equal(t, tc.output, spec)
		})
	}
}
