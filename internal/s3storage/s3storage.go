package s3storage

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/tracing"
)

type Service struct {
	bucketName string
	s3client   *s3.S3
	tracer     tracing.Tracer
}

func New(tracer tracing.Tracer) (*Service, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-west-1"),
	})
	if err != nil {
		return nil, err
	}
	// Create a S3 client from just a session.
	s3client := s3.New(sess)

	bucketName := "lunar-release-artifacts"

	return &Service{
		bucketName: bucketName,
		s3client:   s3client,
		tracer:     tracer,
	}, nil
}

func (s *Service) InitializeBucket() error {
	_, err := s.s3client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(s.bucketName),
	})
	aerr, isAwsErr := err.(awserr.Error)
	if isAwsErr && (aerr.Code() == s3.ErrCodeBucketAlreadyOwnedByYou || aerr.Code() == s3.ErrCodeBucketAlreadyExists) {
		log.WithFields("type", "s3storage").Info("s3 bucket already exists")
		return nil
	}
	if err != nil {
		return err
	}
	log.WithFields("type", "s3storage").Info("s3 bucket create")
	return nil
}

func (s *Service) CreateArtifact(artifactSpec artifact.Spec) (string, error) {
	metadataSpec, err := EncodeSpecToMetadataContent(artifactSpec)
	if err != nil {
		return "", err
	}

	req, _ := s.s3client.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(getObjectKeyName(artifactSpec.Service, artifactSpec.ID)),
		Metadata: map[string]*string{
			MetadataArtifactSpecPartialWriteKey: aws.String(metadataSpec),
		},
	})
	// TODO: Add MD5 content hashing
	// h := md5.New()
	// md5s := base64.StdEncoding.EncodeToString(h.Sum(nil))
	// req.HTTPRequest.Header.Set("Content-MD5", md5s)

	uploadURL, err := req.Presign(15 * time.Minute)
	if err != nil {
		return "", err
	}

	return uploadURL, nil
}
