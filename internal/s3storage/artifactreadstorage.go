package s3storage

import (
	"bytes"
	"context"
	"os"
	"path"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

func (f *Service) ArtifactExists(ctx context.Context, service, artifactID string) (bool, error) {
	key := getObjectKeyName(service, artifactID)
	_, err := f.s3client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(f.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
			return false, nil
		}
		return false, errors.Wrapf(err, "head object at key '%s'", key)
	}
	return true, nil
}

func (f *Service) ArtifactSpecification(ctx context.Context, service string, artifactID string) (artifact.Spec, error) {
	return f.getArtifactSpecFromObjectKey(ctx, getObjectKeyName(service, artifactID))
}

func (f *Service) ArtifactPaths(ctx context.Context, service string, environment string, branch string, artifactID string) (specPath string, resourcesPath string, close func(context.Context), err error) {
	key := getObjectKeyName(service, artifactID)
	artifact, close, err := f.downloadArtifact(ctx, key)
	if err != nil {
		return "", "", nil, errors.WithMessagef(err, "download from key '%s'", key)
	}
	resourcesPath, err = securejoin.SecureJoin(artifact, environment)
	if err != nil {
		return "", "", nil, errors.WithMessagef(err, "resources path invalid for '%s'", environment)
	}
	return path.Join(artifact, "artifact.json"), resourcesPath, close, nil
}

func (f *Service) LatestArtifactSpecification(ctx context.Context, service string, branch string) (artifact.Spec, error) {
	key, err := f.getLatestObjectKey(ctx, service, branch)

	if err != nil {
		return artifact.Spec{}, errors.WithMessage(err, "get latest object key")
	}
	log.WithContext(ctx).WithFields("key", key).Infof("Latest artifact for service '%s' and branch '%s' is at key '%s'", service, branch, key)
	return f.getArtifactSpecFromObjectKey(ctx, key)
}

func (f *Service) LatestArtifactPaths(ctx context.Context, service string, environment string, branch string) (specPath string, resourcesPath string, close func(context.Context), err error) {
	key, err := f.getLatestObjectKey(ctx, service, branch)
	if err != nil {
		return "", "", nil, errors.WithMessage(err, "get latest artifact key")
	}
	artifact, close, err := f.downloadArtifact(ctx, key)
	if err != nil {
		return "", "", nil, errors.WithMessagef(err, "download from key '%s'", key)
	}
	resourcesPath, err = securejoin.SecureJoin(artifact, environment)
	if err != nil {
		return "", "", nil, errors.WithMessagef(err, "resources path invalid for '%s'", environment)
	}
	return path.Join(artifact, "artifact.json"), resourcesPath, close, nil
}

func (f *Service) ArtifactSpecifications(ctx context.Context, service string, n int, branch string) ([]artifact.Spec, error) {

	var prefix string

	if branch != "" {
		prefix = getServiceAndBranchObjectKeyPrefix(service, branch)
	} else {
		prefix = getServiceObjectKeyPrefix(service)
	}

	list := []*s3.Object{}

	err := f.s3client.ListObjectsV2PagesWithContext(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(f.bucketName),
		MaxKeys: aws.Int64(1000),
		Prefix:  aws.String(prefix),
	}, func(p *s3.ListObjectsV2Output, lastPage bool) bool {
		list = append(list, p.Contents...)
		return true
	})
	if err != nil {
		return nil, errors.Wrapf(err, "list objects with prefix '%s'", prefix)
	}

	log.WithContext(ctx).WithFields("service", service, "count", n).Infof("Found %d artifacts for service '%s'", len(list), service)

	sort.Slice(list, func(i, j int) bool {
		return list[i].LastModified.After(*list[j].LastModified)
	})

	var artifactSpecs []artifact.Spec
	for _, object := range list {
		artifactSpec, err := f.getArtifactSpecFromObjectKey(ctx, *object.Key)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed getting object %s", *object.Key)
		}
		if branch != "" && artifactSpec.Application.Branch != branch {
			continue
		}
		artifactSpecs = append(artifactSpecs, artifactSpec)
		if len(artifactSpecs) >= n {
			break
		}
	}

	return artifactSpecs, nil
}

func (f *Service) getArtifactSpecFromObjectKey(ctx context.Context, objectKey string) (artifact.Spec, error) {
	span, ctx := f.tracer.FromCtx(ctx, "s3storage.getArtifactSpecFromObjectKey")
	defer span.Finish()

	artifactPath, close, err := f.downloadArtifact(ctx, objectKey)
	if err != nil {
		return artifact.Spec{}, errors.WithMessagef(err, "download from key '%s'", objectKey)
	}
	defer close(ctx)

	subSpan, _ := f.tracer.FromCtx(ctx, "read json file")
	defer subSpan.Finish()
	jsonSpec, err := os.ReadFile(path.Join(artifactPath, "artifact.json"))
	if err != nil {
		return artifact.Spec{}, errors.WithMessage(err, "read artifact.json file")
	}

	artifactSpec, err := artifact.Decode(bytes.NewReader(jsonSpec))
	if err != nil {
		return artifact.Spec{}, errors.WithMessage(err, "decode artifact spec")
	}

	return artifactSpec, nil
}
