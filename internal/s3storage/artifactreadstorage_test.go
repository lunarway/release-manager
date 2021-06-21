package s3storage_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
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
			s3Client := EnsureTestS3Objects(t, tc.s3)
			svc, err := s3storage.New(tc.s3.BucketName, s3Client, nil, tracing.NewNoop())
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
	newArtifact := func(service, id, branch string) artifact.Spec {
		return artifact.Spec{
			ID:      id,
			Service: service,
			Application: artifact.Repository{
				Branch: branch,
			},
		}
	}
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
			log.Init(&log.Configuration{
				Level: log.Level{
					Level: zapcore.DebugLevel,
				},
				Development: true,
			})

			bucketName := "a-bucket"
			s3Mock, close := setupS3(t, bucketName, tc.objects...)
			defer close()

			svc, err := s3storage.New(bucketName, s3Mock, nil, tracing.NewNoop())
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

// setupS3 instantiates a fake S3 backend and seeds a bucket with provided
// artifacts.
func setupS3(t *testing.T, bucket string, artifacts ...artifact.Spec) (s3iface.S3API, func()) {
	t.Helper()
	backend := s3mem.New()
	faker := gofakes3.New(backend)
	ts := httptest.NewServer(faker.Server())

	// configure S3 client
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials("YOUR-ACCESSKEYID", "YOUR-SECRETACCESSKEY", ""),
		Endpoint:         aws.String(ts.URL),
		Region:           aws.String("eu-central-1"),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	}
	newSession, err := session.NewSession(s3Config)
	if err != nil {
		t.Fatalf("Failed to instantiate session: %v", err)
	}

	s3Client := s3.New(newSession)

	// Create a new bucket using the CreateBucket call.
	_, err = s3Client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatalf("Failed to create test bucket: %v", err)
	}

	seedArtifacts(t, s3Client, bucket, artifacts)

	return s3Client, ts.Close
}

func seedArtifacts(t *testing.T, s3Client s3iface.S3API, bucket string, artifacts []artifact.Spec) {
	t.Helper()
	for _, artifact := range artifacts {
		artifactZip := zipArtifact(t, artifact)

		_, err := s3Client.PutObject(&s3.PutObjectInput{
			Body:   strings.NewReader(artifactZip),
			Bucket: aws.String(bucket),

			Key: aws.String(fmt.Sprintf("%s/%s", artifact.Service, artifact.ID)),
		})
		if err != nil {
			t.Fatalf("Failed to marshal artifact: %v", err)
		}
		t.Logf("Uploaded test artifact %s to bucket", artifact.ID)

		// sleep to ensure we get different modified timestamps in S3
		time.Sleep(2 * time.Millisecond)
	}
}

func zipArtifact(t *testing.T, artifact artifact.Spec) string {
	t.Helper()
	artifactJSON, err := json.Marshal(artifact)
	if err != nil {
		t.Fatalf("Failed to marshal artifact: %v", err)
	}

	var zipBytes bytes.Buffer
	zipWriter := zip.NewWriter(&zipBytes)
	if err != nil {
		t.Fatalf("Failed to instantiate zip writer: %v", err)
	}
	zipFile, err := zipWriter.Create("artifact.json")
	if err != nil {
		t.Fatalf("Failed to create zip file: %v", err)
	}
	_, err = zipFile.Write(artifactJSON)
	if err != nil {
		t.Fatalf("Failed to write artifact to zip file: %v", err)
	}
	err = zipWriter.Close()
	if err != nil {
		t.Fatalf("Failed to close zip file: %v", err)
	}
	return zipBytes.String()
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
			s3Client := EnsureTestS3Objects(t, tc.s3)
			svc, err := s3storage.New(tc.s3.BucketName, s3Client, nil, tracing.NewNoop())
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
			s3Client := EnsureTestS3Objects(t, tc.s3)
			svc, err := s3storage.New(tc.s3.BucketName, s3Client, nil, tracing.NewNoop())
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
