package s3storage

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/pkg/errors"
)

type Service struct {
	region      string
	bucketName  string
	s3client    *s3.S3
	sqsClient   *sqs.SQS
	tracer      tracing.Tracer
	sqsQueueURL string
	sqsQueueARN string
}

func New(bucketName string, tracer tracing.Tracer) (*Service, error) {
	region := "eu-west-1"
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, err
	}

	s3client := s3.New(sess)
	sqsClient := sqs.New(sess)

	return &Service{
		region:     region,
		bucketName: bucketName,
		s3client:   s3client,
		sqsClient:  sqsClient,
		tracer:     tracer,
	}, nil
}

func (s *Service) InitializeSQS() error {
	// Amazon SQS returns only an error if the request includes attributes whose values differ from those of the existing queue.
	queue, err := s.sqsClient.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(s.sqsQueueName()),
	})
	if err != nil {
		return errors.Wrap(err, "create sqs queue")
	}
	log.Infof("Created queue with url: %s", *queue.QueueUrl)
	s.sqsQueueURL = *queue.QueueUrl

	queueAttributes, err := s.sqsClient.GetQueueAttributes(&sqs.GetQueueAttributesInput{
		QueueUrl:       queue.QueueUrl,
		AttributeNames: []*string{aws.String(sqs.QueueAttributeNameQueueArn)},
	})
	if err != nil {
		return errors.Wrap(err, "get sqs queue ARN")
	}
	s.sqsQueueARN = *queueAttributes.Attributes[sqs.QueueAttributeNameQueueArn]
	log.Infof("SQS queue arn acquired: %s", s.sqsQueueARN)

	policy := `{
		"Version": "2012-10-17",
		"Id": "` + s.sqsQueueARN + `/SQSDefaultPolicy",
		"Statement": [
		 {
			"Sid": "` + s.bucketName + `-s3-permission",
			"Effect": "Allow",
			"Principal": {
			 "AWS":"*"
			},
			"Action": [
			 "SQS:SendMessage"
			],
			"Resource": "` + s.sqsQueueARN + `",
			"Condition": {
				 "ArnLike": { "aws:SourceArn": "arn:aws:s3:*:*:` + s.bucketName + `" }
			}
		 }
		]
	 }`

	_, err = s.sqsClient.SetQueueAttributes(&sqs.SetQueueAttributesInput{
		QueueUrl: &s.sqsQueueURL,
		Attributes: aws.StringMap(map[string]string{
			sqs.QueueAttributeNamePolicy: policy,
		}),
	})
	if err != nil {
		log.With("policy", policy, "bucketName", s.bucketName, "Error", fmt.Sprintf("%s", err)).Errorf("Failed update policy: %s", err)
		log.Errorf("Failed with body: %s", err)
		return errors.Wrap(err, "update sqs permission")
	}
	log.Infof("SQS policy updated for s3 bucket %s to SQS %s", s.bucketName, s.sqsQueueARN)

	go func() {
		for {
			log.Infof("Looping messages")
			output, err := s.sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(s.sqsQueueURL),
				MaxNumberOfMessages: aws.Int64(1),
				WaitTimeSeconds:     aws.Int64(5),
			})

			if err != nil {
				log.Errorf("failed to fetch sqs message %v", err)
			}

			for _, message := range output.Messages {
				log.Infof("Received message %s: %#v", *message.MessageId, *message.Body)
				_, err := s.sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
					QueueUrl:      &s.sqsQueueURL,
					ReceiptHandle: message.ReceiptHandle,
				})
				if err != nil {
					log.Errorf("Failed deleting SQS message %s. Error: %s", *message.ReceiptHandle, err)
				}
			}
		}
	}()

	return nil
}

func (s *Service) InitializeBucket() error {
	_, err := s.s3client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(s.bucketName),
	})
	aerr, isAwsErr := err.(awserr.Error)
	if isAwsErr && (aerr.Code() == s3.ErrCodeBucketAlreadyOwnedByYou || aerr.Code() == s3.ErrCodeBucketAlreadyExists) {
		log.WithFields("type", "s3storage").Info("s3 bucket already exists")
	} else if err != nil {
		return errors.Wrap(err, "create bucket")
	}
	log.WithFields("type", "s3storage").Info("s3 bucket ensured")

	_, err = s.s3client.PutBucketNotificationConfiguration(&s3.PutBucketNotificationConfigurationInput{
		Bucket: aws.String(s.bucketName),
		NotificationConfiguration: &s3.NotificationConfiguration{
			QueueConfigurations: []*s3.QueueConfiguration{
				&s3.QueueConfiguration{
					QueueArn: aws.String(s.sqsQueueARN),
					Events: []*string{
						aws.String(s3.EventS3ObjectCreated),
					},
				},
			},
		},
	})
	if err == nil {
		return errors.Wrap(err, "update bucket notifications")
	}
	log.WithFields("type", "s3storage").Info("s3 bucket notifications updated")

	return nil
}

func (s *Service) CreateArtifact(artifactSpec artifact.Spec, md5 string) (string, error) {
	key := getObjectKeyName(artifactSpec.Service, artifactSpec.ID)

	req, _ := s.s3client.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	req.HTTPRequest.Header.Set("Content-MD5", md5)

	uploadURL, err := req.Presign(15 * time.Minute)
	if err != nil {
		return "", errors.Wrapf(err, "create put object for key '%s'", key)
	}

	return uploadURL, nil
}

func (s *Service) sqsQueueName() string {
	return fmt.Sprintf("%s-s3-bucket-notifications", s.bucketName)
}

// func (s *Service) sqsQueueArn() string {
// 	https://sqs.us-east-2.amazonaws.com/123456789012/MyQueue

// 	re := regexp.MustCompile(^`https://sqs.(?P<region>[^.]+).amazonaws.com/(?P<accountnumber>[^/]+)/(?P<queuename>.*)$`)
// 	fmt.Println(re.MatchString(s.sqsQueueURL))
// 	fmt.Printf("%q\n", re.SubexpNames())
// 	reversed := fmt.Sprintf("${%s} ${%s}", re.SubexpNames()[2], re.SubexpNames()[1])
// 	fmt.Println(reversed)
// 	fmt.Println(re.ReplaceAllString("Alan Turing", reversed))

// 	return fmt.Sprintf("arn:aws:sqs:%s:%s:%s", region, "", s.sqsQueueName())
// }
