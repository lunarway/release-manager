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
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/assert"
)

func SkipIfNoAWS(t *testing.T) {
	if os.Getenv("AWS_CONFIG_FILE") == "" {
		t.Skip("AWS not configured to run AWS integration tests")
	}
}

func EnsureTestS3Objects(t *testing.T, setup S3BucketSetup) s3iface.S3API {
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
		t.Logf("s3 bucket already exists")
		err = emptyBucket(t, s3client, setup.BucketName)
		assert.NoError(t, err)
	} else {
		assert.NoError(t, err)
	}

	t.Log("s3 bucket ensured")
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
	return s3client
}

type S3BucketSetup struct {
	BucketName string
	Objects    map[string]S3BucketSetupObject
}

type S3BucketSetupObject struct {
	Metadata      map[string]string
	Base64Content string
}

func emptyBucket(t *testing.T, s3client *s3.S3, bucket string) error {
	t.Logf("removing objects from S3 bucket: %s", bucket)
	for {
		params := &s3.ListObjectsInput{
			Bucket: aws.String(bucket),
		}

		objects, err := s3client.ListObjects(params)
		if err != nil {
			return err
		}
		//Checks if the bucket is already empty
		if len((*objects).Contents) == 0 {
			t.Log("Bucket is already empty")
			return nil
		}
		t.Logf("First object in batch %s", *(objects.Contents[0].Key))

		//creating an array of pointers of ObjectIdentifier
		objectsToDelete := make([]*s3.ObjectIdentifier, 0, 1000)
		for _, object := range (*objects).Contents {
			obj := s3.ObjectIdentifier{
				Key: object.Key,
			}
			objectsToDelete = append(objectsToDelete, &obj)
		}
		//Creating JSON payload for bulk delete
		deleteArray := s3.Delete{Objects: objectsToDelete}
		deleteParams := &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &deleteArray,
		}
		//Running the Bulk delete job (limit 1000)
		_, err = s3client.DeleteObjects(deleteParams)
		if err != nil {
			return err
		}
		if *(*objects).IsTruncated { //if there are more objects in the bucket, IsTruncated = true
			params.Marker = (*deleteParams).Delete.Objects[len((*deleteParams).Delete.Objects)-1].Key
			t.Logf("Requesting next batch %s", *(params.Marker))
		} else { //if all objects in the bucket have been cleaned up.
			break
		}
	}
	t.Logf("Emptied S3 bucket %s", bucket)
	return nil
}
