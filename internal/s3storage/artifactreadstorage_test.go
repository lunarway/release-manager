package s3storage_test

import (
	"context"
	"testing"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/s3storage"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

var _ flow.ArtifactReadStorage = &s3storage.Service{}

func TestService_ArtifactPaths(t *testing.T) {
	SkipIfNoAWS(t)
	tt := []struct {
		name       string
		service    string
		env        string
		branch     string
		artifactID string
		s3         S3BucketSetup
	}{
		{
			name:       "known artifact",
			service:    "test-service",
			env:        "dev",
			artifactID: "master-1234ds13g3-12s46g356g",
			s3: S3BucketSetup{
				BucketName: "release-manager-test",
				Objects: map[string]S3BucketSetupObject{
					"test-service/master-1234ds13g3-12s46g356g": {
						Base64Content: S3File_ZippedArtifact,
					},
				},
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			log.Init(&log.Configuration{
				Level: log.Level{
					Level: zapcore.DebugLevel,
				},
				Development: true,
			})
			EnsureTestS3Objects(t, tc.s3)
			svc, err := s3storage.New(tc.s3.BucketName, tracing.NewNoop())
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

func TestService_ArtifactSpecification(t *testing.T) {
	SkipIfNoAWS(t)
	tt := []struct {
		name       string
		service    string
		artifactID string
		s3         S3BucketSetup
	}{
		{
			name:       "known artifact",
			service:    "test-service",
			artifactID: "master-1234ds13g3-12s46g356g",
			s3: S3BucketSetup{
				BucketName: "release-manager-test-artifact-specification",
				Objects: map[string]S3BucketSetupObject{
					"test-service/master-1234ds13g3-12s46g356g": {
						Base64Content: S3File_ZippedArtifact,
					},
				},
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			log.Init(&log.Configuration{
				Level: log.Level{
					Level: zapcore.DebugLevel,
				},
				Development: true,
			})
			EnsureTestS3Objects(t, tc.s3)
			svc, err := s3storage.New(tc.s3.BucketName, tracing.NewNoop())
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
