package s3storage_test

import (
	"context"
	"encoding/base64"
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
						Metadata: map[string]string{
							"artifact-spec": "",
						},
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
			artifactID: "master-123-123",
			s3: S3BucketSetup{
				BucketName: "release-manager-test-artifact-specification",
				Objects: map[string]S3BucketSetupObject{
					"test-service/master-123-123": {
						Base64Content: S3File_Empty,
						Metadata: map[string]string{
							"artifact-spec": base64.StdEncoding.EncodeToString([]byte(("{\"id\":\"master-123-123\",\"service\":\"test-service\",\"application\":{\"branch\":\"master\",\"sha\":\"asd39sdas0g392\",\"authorName\":\"Kasper Nissen\",\"authorEmail\":\"kni@lunar.app\",\"committerName\":\"Bjørn Sørensen\",\"committerEmail\":\"bso@lunar.app\",\"message\":\"Some message\",\"name\":\"lunar-way-application\",\"url\":\"https://someurl.com\",\"provider\":\"BitBucket\"},\"ci\":{\"jobUrl\":\"https://jenkins.dev.lunarway.com/job/asdasd\",\"start\":\"2020-05-29T15:32:16.780238+02:00\",\"end\":\"2020-05-29T15:32:16.92275+02:00\"},\"shuttle\":{\"plan\":{\"branch\":\"plan-branch\",\"sha\":\"asdasdo300asd0asd90as92\",\"message\":\"Some commit\",\"url\":\"https://someplanurl\"}},\"stages\":[ {\"id\":\"build\",\"name\":\"Build\",\"data\":{\"dockerVersion\":\"1.18.6\",\"image\":\"quay.io/lunarway/application\",\"tag\":\"master-1234ds13g3-12s46g356g\"}},{\"id\":\"push\",\"name\":\"Push\",\"data\":{\"dockerVersion\":\"1.18.6\",\"image\":\"quay.io/lunarway/application\",\"tag\":\"master-1234ds13g3-12s46g356g\"}},{\"id\":\"test\",\"name\":\"Test\",\"data\":{\"results\":{\"failed\":0,\"passed\":563,\"skipped\":0},\"url\":\"https://jenkins.dev.lunarway.com\"}},{\"id\":\"snyk-code\",\"name\":\"Security Scan - Code\",\"data\":{\"language\":\"go\",\"snykVersion\":\"1.144.23\",\"url\":\"https://snyk.io/aslkdasdlas\",\"vulnerabilities\":{\"high\":2,\"low\":134,\"medium\":23}}},{\"id\":\"snyk-docker\",\"name\":\"Security Scan - Docker\",\"data\":{\"baseImage\":\"node\",\"snykVersion\":\"1.144.23\",\"tag\":\"8.15.0-alpine\",\"url\":\"https://snyk.io/aslkdasdlas\",\"vulnerabilities\":{\"high\":0,\"low\":0,\"medium\":0}}} ]}"))),
						},
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
