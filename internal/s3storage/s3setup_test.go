package s3storage_test

import (
	"bytes"
	"encoding/base64"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
)

func SkipIfNoAWS(t *testing.T) {
	if os.Getenv("AWS_CONFIG_FILE") == "" {
		t.Skip("AWS not configured to run AWS integration tests")
	}
}

func EnsureTestS3Objects(t *testing.T, setup S3BucketSetup) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-west-1"),
	})
	s3client := s3.New(sess)
	assert.NoError(t, err)
	_, err = s3client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(setup.BucketName),
	})
	aerr, isAwsErr := err.(awserr.Error)
	if isAwsErr && (aerr.Code() == s3.ErrCodeBucketAlreadyOwnedByYou || aerr.Code() == s3.ErrCodeBucketAlreadyExists) {
		t.Log("s3 bucket already exists")
	} else {
		assert.NoError(t, err)
	}
	t.Log("s3 bucket created")
	for keyName, object := range setup.Objects {

		objectContent, err := base64.StdEncoding.DecodeString(object.Base64Content)
		assert.NoError(t, err)

		_, err = s3client.PutObject(&s3.PutObjectInput{
			Bucket:   aws.String(setup.BucketName),
			Key:      aws.String(keyName),
			Body:     bytes.NewReader(objectContent),
			Metadata: aws.StringMap(object.Metadata),
		})
		assert.NoError(t, err)
	}
}

type S3BucketSetup struct {
	BucketName string
	Objects    map[string]S3BucketSetupObject
}

type S3BucketSetupObject struct {
	Metadata      map[string]string
	Base64Content string
}
