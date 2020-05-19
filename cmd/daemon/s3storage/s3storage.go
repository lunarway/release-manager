package s3storage

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func Initialize() error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-west-1"),
	})
	if err != nil {
		return err
	}
	// Create a S3 client from just a session.
	s3client := s3.New(sess)

	bucketName := aws.String("lunar-release-artifacts-2")

	err = createBucket(s3client, bucketName)
	if err != nil {
		return err
	}
	// _, err = s3client.HeadBucket(&s3.HeadBucketInput{
	// 	Bucket: bucketName,
	// })
	// aerr, isAwsErr := err.(awserr.Error)
	// switch {
	// case isAwsErr && aerr.Code() == s3.ErrCodeNoSuchBucket:
	// 	createBucket(s3client, bucketName)
	// case isAwsErr:
	// 	fmt.Println(aerr.Error())
	// 	return err
	// case err != nil:
	// 	return err
	// }

	return nil
}

func createBucket(s3client *s3.S3, bucketName *string) error {
	result, err := s3client.CreateBucket(&s3.CreateBucketInput{
		Bucket: bucketName,
	})
	aerr, isAwsErr := err.(awserr.Error)
	if isAwsErr && (aerr.Code() == s3.ErrCodeBucketAlreadyOwnedByYou || aerr.Code() == s3.ErrCodeBucketAlreadyExists) {
		fmt.Println("Bucket already exists")
		return nil
	}
	if err != nil {
		return err
	}

	fmt.Println(result)
	return nil
}
