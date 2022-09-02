package flow

import (
	"context"
	"testing"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/copy"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
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
				ID: "master-default-5678",
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

			logger := log.New(&log.Configuration{
				Level: log.Level{
					Level: zapcore.DebugLevel,
				},
				Development: true,
			})

			s := Service{
				Copier:           copy.New(logger),
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

func TestReleaseSpecifications(t *testing.T) {
	tt := []struct {
		name      string
		namespace string
		service   string
		release   ReleaseSpec
	}{
		{
			name:      "default namespace",
			namespace: "",
			service:   "a",
			release: ReleaseSpec{
				Environment: "dev",
				Spec: artifact.Spec{
					ID: "master-default-5678",
				},
			},
		},
		{
			name:      "specified namespace",
			namespace: "other",
			service:   "a",
			release: ReleaseSpec{
				Environment: "dev",
				Spec: artifact.Spec{
					ID: "master-other-5678",
				},
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			gitService := MockGitService{}
			gitService.Test(t)
			gitService.On("MasterPath").Return("testdata")

			logger := log.New(&log.Configuration{
				Level: log.Level{
					Level: zapcore.DebugLevel,
				},
				Development: true,
			})

			s := Service{
				Copier:           copy.New(logger),
				Git:              &gitService,
				ArtifactFileName: "artifact.json",
			}

			releases, err := s.releaseSpecifications(context.Background(), tc.namespace, tc.service)

			assert.NoError(t, err, "unexpected error")
			assert.Equal(t, []ReleaseSpec{tc.release}, releases, "release specs not as expected")
		})
	}
}
