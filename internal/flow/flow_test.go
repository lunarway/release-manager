package flow

import (
	"context"
	"testing"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestReleaseSpecification(t *testing.T) {
	tt := []struct {
		name     string
		location releaseLocation
		err      error
		spec     artifact.Spec
	}{
		{
			name: "sunshine",
			location: releaseLocation{
				Environment: "dev",
				Namespace:   "dev",
				Service:     "a",
			},
			spec: artifact.Spec{
				ID: "master-1234-5678",
			},
		},
		{
			name: "path traversal vulnerability",
			location: releaseLocation{
				Environment: "dev",
				Namespace:   "dev",
				Service:     "../",
			},
			spec: artifact.Spec{},
			err:  artifact.ErrFileNotFound,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			gitService := MockGitService{}
			gitService.Test(t)
			gitService.On("MasterPath").Return("testdata")

			s := Service{
				Git:              &gitService,
				ArtifactFileName: "artifact.json",
			}

			spec, err := s.releaseSpecification(context.Background(), tc.location)
			if tc.err != nil {
				assert.EqualError(t, errors.Cause(err), tc.err.Error(), "error not as expected")
			} else {
				assert.NoError(t, err, "unexpected error")
			}
			assert.Equal(t, tc.spec, spec, "artifact spec not as expected")
		})
	}
}
