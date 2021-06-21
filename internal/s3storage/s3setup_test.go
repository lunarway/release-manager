package s3storage_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
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
)

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
