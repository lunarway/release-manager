package s3storage

import (
	"context"
	"fmt"
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
	defer span.End()
	prefix := getServiceAndBranchObjectKeyPrefix(service, branch)
	var latest *s3.Object
	err := f.s3client.ListObjectsV2PagesWithContext(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(f.bucketName),
		Prefix: aws.String(prefix),
	}, func(page *s3.ListObjectsV2Output, _ bool) bool {
		for _, object := range page.Contents {
			// Artifact IDs are {branch}-{appSha}-{planSha}, so keys with more
			// segments after the prefix belong to branches starting with the
			// requested branch name followed by '-', e.g. feature-x for feature.
			if strings.Count(strings.TrimPrefix(*object.Key, prefix), "-") != 1 {
				continue
			}
			if latest == nil || object.LastModified.After(*latest.LastModified) {
				latest = object
			}
		}
		return true
	})
	if err != nil {
		return "", errors.Wrapf(err, "list objects at prefix '%s'", prefix)
	}

	if latest == nil {
		return "", flow.ErrArtifactNotFound
	}

	return *latest.Key, nil
}
