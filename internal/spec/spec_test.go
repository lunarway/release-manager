package spec_test

import (
	"path"
	"testing"

	"github.com/lunarway/release-manager/internal/spec"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	type output struct {
	}
	tt := []struct {
		name string
		path string
		spec spec.Spec
		err  error
	}{
		{
			name: "existing and valid artifact",
			path: "valid_artifact.json",
			spec: spec.Spec{
				ID: "valid",
			},
		},
		{
			name: "unknown artifact",
			path: "unknown_artifact.json",
			spec: spec.Spec{},
			err:  spec.ErrFileNotFound,
		},
		{
			name: "invalid artifact",
			path: "invalid_artifact.json",
			spec: spec.Spec{},
			err:  spec.ErrNotParsable,
		},
		{
			name: "unknown fields in artifact",
			path: "unknown_fields_artifact.json",
			spec: spec.Spec{},
			err:  spec.ErrUnknownFields,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := spec.Get(path.Join("testdata", tc.path))
			if tc.err != nil {
				assert.EqualError(t, errors.Cause(err), tc.err.Error(), "output error not as expected")
			} else {
				assert.NoError(t, err, "no output error expected")
			}
			assert.Equal(t, tc.spec, spec, "spec not as expected")
		})
	}
}
