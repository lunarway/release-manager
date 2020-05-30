package s3storage

import (
	"context"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func getObjectKeyName(service string, artifactID string) string {
	return fmt.Sprintf("%s/%s", service, artifactID)
}
func getServiceObjectKeyPrefix(service string) string {
	return fmt.Sprintf("%s/", service)
}
func getServiceAndBranchObjectKeyPrefix(service, branch string) string {
	return fmt.Sprintf("%s/%s-", service, branch)
}

func (f *Service) getLatestObjectKey(ctx context.Context, service string, branch string) (string, error) {
	list, err := f.s3client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:  aws.String(f.bucketName),
		MaxKeys: aws.Int64(1000), // TODO: Find a solution to handle more than 1000
		Prefix:  aws.String(getServiceAndBranchObjectKeyPrefix(service, branch)),
	})

	if err != nil {
		return "", err
	}

	sort.Slice(list.Contents, func(i, j int) bool {
		return list.Contents[i].LastModified.After(*list.Contents[j].LastModified)
	})

	return *list.Contents[0].Key, nil
}
