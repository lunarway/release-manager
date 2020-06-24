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

func TestService_LatestArtifactSpecification(t *testing.T) {
	SkipIfNoAWS(t)
	tt := []struct {
		name             string
		service          string
		branch           string
		s3               S3BucketSetup
		expectedArtifact artifact.Spec
		expectedError    string
	}{
		{
			name:    "no artifacts",
			service: "test-service",
			branch:  "master",
			s3: S3BucketSetup{
				BucketName: "release-manager-test-latest-artifact-specification",
				Objects:    map[string]S3BucketSetupObject{},
			},
			expectedError: "get latest object key: artifact not found",
		},
		{
			name:    "no artifacts",
			service: "test-service",
			branch:  "master",
			s3: S3BucketSetup{
				BucketName: "release-manager-test-latest-artifact-specification",
				Objects: map[string]S3BucketSetupObject{
					"test-service/master-1234ds13g3-12s46g356g": {
						Base64Content: S3File_ZippedArtifact,
					},
				},
			},
			expectedArtifact: artifact.Spec{
				ID: "master-1234ds13g3-12s46g356g",
			},
		},
		{
			name:    "works with features branches",
			service: "test-service",
			branch:  "feature/greatwork",
			s3: S3BucketSetup{
				BucketName: "release-manager-test-latest-artifact-specification",
				Objects: map[string]S3BucketSetupObject{
					"test-service/feature_greatwork-1234ds13g3-12s46g356g": {
						Base64Content: RewriteArtifactWithSpec(S3File_ZippedArtifact, func(spec *artifact.Spec) {
							spec.ID = "feature_greatwork-1234ds13g3-12s46g356g"
							spec.Application.Branch = "feature/greatwork"
						}),
					},
				},
			},
			expectedArtifact: artifact.Spec{
				ID: "feature_greatwork-1234ds13g3-12s46g356g",
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
