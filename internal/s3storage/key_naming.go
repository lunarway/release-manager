package s3storage

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/pkg/errors"
)

func getObjectKeyName(service string, artifactID string) string {
	return fmt.Sprintf("%s/%s", service, artifactID)
}
func getServiceObjectKeyPrefix(service string) string {
	return fmt.Sprintf("%s/", service)
}
func getServiceAndBranchObjectKeyPrefix(service, branch string) string {

	return fmt.Sprintf("%s/%s-", service, strings.ReplaceAll(branch, "/", "_"))
}

func (f *Service) getLatestObjectKey(ctx context.Context, service string, branch string) (string, error) {
	span, ctx := f.tracer.FromCtx(ctx, "s3storage.getLatestObjectKey")
	defer span.Finish()
	prefix := getServiceAndBranchObjectKeyPrefix(service, branch)
	list, err := f.s3client.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(f.bucketName),
		MaxKeys: aws.Int64(1000), // TODO: Find a solution to handle more than 1000
		Prefix:  aws.String(prefix),
	})

	if err != nil {
		return "", errors.Wrapf(err, "list objects at prefix '%s'", prefix)
	}

	sort.Slice(list.Contents, func(i, j int) bool {
		return list.Contents[i].LastModified.After(*list.Contents[j].LastModified)
	})

	if len(list.Contents) == 0 {
		return "", flow.ErrArtifactNotFound
	}

	return *list.Contents[0].Key, nil
}
