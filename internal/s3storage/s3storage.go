package s3storage

import (
	"crypto/md5"
	"encoding/base64"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/log"
)

type Service struct {
	bucketName string
	logger     *log.Logger
	s3client   *s3.S3
}

func Initialize(logger *log.Logger) (*Service, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-west-1"),
	})
	if err != nil {
		return nil, err
	}
	// Create a S3 client from just a session.
	s3client := s3.New(sess)

	bucketName := "lunar-release-artifacts"

	err = createBucket(s3client, bucketName, logger)
	if err != nil {
		return nil, err
	}

	return &Service{
		bucketName: bucketName,
		logger:     logger,
		s3client:   s3client,
	}, nil
}

func (s *Service) CreateArtifact(artifactSpec artifact.Spec) (string, error) {
	jsonSpec, err := artifact.Encode(artifactSpec, false)
	if err != nil {
		return "", err
	}

	req, _ := s.s3client.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(artifactSpec.ID),
		Metadata: map[string]*string{
			"ArtifactSpec": aws.String(jsonSpec),
		},
	})
	h := md5.New()
	md5s := base64.StdEncoding.EncodeToString(h.Sum(nil))
	req.HTTPRequest.Header.Set("Content-MD5", md5s)

	uploadURL, err := req.Presign(15 * time.Minute)
	if err != nil {
		return "", err
	}

	return uploadURL, nil
}

func createBucket(s3client *s3.S3, bucketName string, logger *log.Logger) error {
	_, err := s3client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	aerr, isAwsErr := err.(awserr.Error)
	if isAwsErr && (aerr.Code() == s3.ErrCodeBucketAlreadyOwnedByYou || aerr.Code() == s3.ErrCodeBucketAlreadyExists) {
		log.Info("s3 bucket already exists")
		return nil
	}
	if err != nil {
		return err
	}
	log.Info("s3 bucket create")
	return nil
}
