package s3storage_test

import (
	"context"
	"sort"
	"testing"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/s3storage"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

var _ flow.ArtifactReadStorage = &s3storage.Service{}

func newArtifact(service, id, branch string) artifact.Spec {
	return artifact.Spec{
		ID:      id,
		Service: service,
		Application: artifact.Repository{
			Branch: branch,
		},
	}
}

func TestService_ArtifactPaths(t *testing.T) {
	tt := []struct {
		name            string
		service         string
		env             string
		branch          string
		artifactID      string
		storedArtifacts []artifact.Spec
	}{
		{
			name:       "known artifact",
			service:    "test-service",
			env:        "dev",
			artifactID: "master-1234ds13g3-12s46g356g",
			storedArtifacts: []artifact.Spec{
				newArtifact("test-service", "master-1234ds13g3-12s46g356g", "master"),
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New(&log.Configuration{
				Level: log.Level{
					Level: zapcore.DebugLevel,
				},
				Development: true,
			})
			bucket := "a-bucket"
			s3Client, s3Close := setupS3(t, bucket, tc.storedArtifacts...)
			defer s3Close()
			svc, err := s3storage.New(bucket, s3Client, nil, tracing.NewNoop(), logger)
			if !assert.NoError(t, err, "initialization error") {
				return
			}
			ctx := context.Background()
			specPath, resourcesPath, close, err := svc.ArtifactPaths(ctx, tc.service, tc.env, tc.branch, tc.artifactID)
			if !assert.NoError(t, err, "get paths error") {
				return
			}
			defer close(ctx)
			t.Logf("Spec in %s", specPath)
			t.Logf("Resources in %s", resourcesPath)
			spec, err := artifact.Get(specPath)
			if !assert.NoError(t, err, "artifact could not be read") {
				return
			}
			t.Logf("Spec: %#v", spec)
			assert.Equal(t, tc.artifactID, spec.ID, "artifact ID not as expected")
		})
	}
}

func TestService_ArtifactSpecifications(t *testing.T) {
	tt := []struct {
		name      string
		service   string
		count     int
		branch    string
		objects   []artifact.Spec
		artifacts []artifact.Spec
	}{
		{
			name:      "no artifacts found",
			service:   "foo",
			count:     10,
			branch:    "",
			artifacts: nil,
		},
		{
			name:    "single artifact found",
			service: "foo",
			count:   10,
			branch:  "",
			objects: []artifact.Spec{
				newArtifact("foo", "master-1-2", "master"),
			},
			artifacts: []artifact.Spec{
				newArtifact("foo", "master-1-2", "master"),
			},
		},
		{
			name:    "no artifacts found matching branch filter",
			service: "foo",
			count:   10,
			branch:  "master",
			objects: []artifact.Spec{
				newArtifact("foo", "feature_awesome-1-2", "feature/awesome"),
			},
			artifacts: nil,
		},
		{
			name:    "mixed artifacts found with filtered branch and count",
			service: "foo",
			count:   2,
			branch:  "master",
			objects: []artifact.Spec{
				newArtifact("foo", "master-1-2", "master"),
				newArtifact("foo", "feature_awesome-1-2", "feature/awesome"),
				newArtifact("foo", "master-2-3", "master"),
				newArtifact("foo", "master-3-4", "master"),
			},
			artifacts: []artifact.Spec{
				newArtifact("foo", "master-2-3", "master"),
				newArtifact("foo", "master-3-4", "master"),
			},
		},
		{
			name:    "more artifacts than count",
			service: "foo",
			count:   1,
			branch:  "",
			objects: []artifact.Spec{
				newArtifact("foo", "master-1-2", "master"),
				newArtifact("foo", "master-2-3", "master"),
			},
			artifacts: []artifact.Spec{
				newArtifact("foo", "master-2-3", "master"),
			},
		},
		{
			name:    "no artifacts for service",
			service: "bar",
			count:   1,
			branch:  "",
			objects: []artifact.Spec{
				newArtifact("foo", "master-1-2", "master"),
				newArtifact("foo", "master-2-3", "master"),
			},
			artifacts: nil,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New(&log.Configuration{
				Level: log.Level{
					Level: zapcore.DebugLevel,
				},
				Development: true,
			})

			bucketName := "a-bucket"
			s3Mock, close := setupS3(t, bucketName, tc.objects...)
			defer close()

			svc, err := s3storage.New(bucketName, s3Mock, nil, tracing.NewNoop(), logger)
			require.NoError(t, err, "initialization error")

			artifacts, err := svc.ArtifactSpecifications(context.Background(), tc.service, tc.count, tc.branch)
			require.NoError(t, err, "get specifications error")

			// ArtifactSpecifications returns the latest artifacts based on S3 upload
			// timestamps. This sorting is used to make it simple to assert on the
			// order.
			sortArtifactsByID(artifacts)
			assert.Equal(t, tc.artifacts, artifacts)
		})
	}
}

func sortArtifactsByID(artifacts []artifact.Spec) {
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].ID < artifacts[j].ID
	})
}

func TestService_ArtifactSpecification(t *testing.T) {
	tt := []struct {
		name            string
		service         string
		artifactID      string
		storedArtifacts []artifact.Spec
	}{
		{
			name:       "known artifact",
			service:    "test-service",
			artifactID: "master-1234ds13g3-12s46g356g",
			storedArtifacts: []artifact.Spec{
				newArtifact("test-service", "master-1234ds13g3-12s46g356g", "master"),
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New(&log.Configuration{
				Level: log.Level{
					Level: zapcore.DebugLevel,
				},
				Development: true,
			})
			bucket := "a-bucket"
			s3Client, close := setupS3(t, bucket, tc.storedArtifacts...)
			defer close()
			svc, err := s3storage.New(bucket, s3Client, nil, tracing.NewNoop(), logger)
			if !assert.NoError(t, err, "initialization error") {
				return
			}
			ctx := context.Background()

			artifactSpec, err := svc.ArtifactSpecification(ctx, tc.service, tc.artifactID)

			assert.NoError(t, err, "get ArtifactSpecification error")
			assert.Equal(t, tc.artifactID, artifactSpec.ID, "artifact ID not as expected")
		})
	}
}

func TestService_LatestArtifactSpecification(t *testing.T) {
	tt := []struct {
		name             string
		service          string
		branch           string
		storedArtifacts  []artifact.Spec
		expectedArtifact artifact.Spec
		expectedError    string
	}{
		{
			name:            "no artifacts",
			service:         "test-service",
			branch:          "master",
			storedArtifacts: nil,
			expectedError:   "get latest object key: artifact not found",
		},
		{
			name:    "single artifact",
			service: "test-service",
			branch:  "master",
			storedArtifacts: []artifact.Spec{
				newArtifact("test-service", "master-1234ds13g3-12s46g356g", "master"),
			},
			expectedArtifact: newArtifact("test-service", "master-1234ds13g3-12s46g356g", "master"),
		},
		{
			name:    "multiple artifacts",
			service: "test-service",
			branch:  "master",
			storedArtifacts: []artifact.Spec{
				newArtifact("test-service", "master-1-2", "master"),
				newArtifact("test-service", "master-2-3", "master"),
			},
			expectedArtifact: newArtifact("test-service", "master-2-3", "master"),
		},
		{
			name:    "works with features branches",
			service: "test-service",
			branch:  "feature/greatwork",
			storedArtifacts: []artifact.Spec{
				newArtifact("test-service", "feature_greatwork-1234ds13g3-12s46g356g", "feature/greatwork"),
			},
			expectedArtifact: newArtifact("test-service", "feature_greatwork-1234ds13g3-12s46g356g", "feature/greatwork"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New(&log.Configuration{
				Level: log.Level{
					Level: zapcore.DebugLevel,
				},
				Development: true,
			})
			bucket := "a-bucket"
			s3Client, close := setupS3(t, bucket, tc.storedArtifacts...)
			defer close()
			svc, err := s3storage.New(bucket, s3Client, nil, tracing.NewNoop(), logger)
			if !assert.NoError(t, err, "initialization error") {
				return
			}
			ctx := context.Background()

			artifactSpec, err := svc.LatestArtifactSpecification(ctx, tc.service, tc.branch)

			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError, "error returned from LatestArtifactSpecification is not as expected")
				return
			}
			assert.NoError(t, err, "get ArtifactSpecification error")
			assert.Equal(t, tc.expectedArtifact.ID, artifactSpec.ID, "artifact ID not as expected")
		})
	}
}
