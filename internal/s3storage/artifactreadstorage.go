package s3storage

import (
	"context"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/pkg/errors"
)

func (f *Service) ArtifactExists(ctx context.Context, service, artifactID string) (bool, error) {
	_, err := f.s3client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(f.bucketName),
		Key:    aws.String(getObjectKeyName(service, artifactID)),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (f *Service) ArtifactSpecification(ctx context.Context, service string, artifactID string) (artifact.Spec, error) {
	return f.getArtifactSpecFromObjectKey(ctx, getObjectKeyName(service, artifactID))
}

func (f *Service) ArtifactPaths(ctx context.Context, service string, environment string, branch string, artifactID string) (specPath string, resourcesPath string, close func(context.Context), err error) {
	logger := log.WithContext(ctx)

	// FIXME:  we should move that out of package git
	zipDestPath, closeSource, err := git.TempDirAsync(ctx, tracing.NewNoop(), "s3-artifact-paths")
	if err != nil {
		return "", "", nil, errors.WithMessage(err, "get temp dir")
	}
	defer closeSource(ctx)
	zipDestPath = path.Join(zipDestPath, "artifact.zip")
	logger.Debugf("Zip destination: %s", zipDestPath)
	zipDest, err := os.Create(zipDestPath)
	if err != nil {
		return "", "", nil, errors.WithMessage(err, "get temp dir")
	}
	defer func() {
		err := zipDest.Close()
		if err != nil {
			logger.Errorf("Failed to close zip destination file: %v", err)
		}
	}()

	key := getObjectKeyName(service, artifactID)
	downloader := s3manager.NewDownloaderWithClient(f.s3client)
	logger.Infof("Downloading object at key '%s'", key)
	n, err := downloader.DownloadWithContext(ctx, zipDest, &s3.GetObjectInput{
		Bucket: aws.String(f.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", "", nil, errors.WithMessage(err, "download object")
	}
	logger.Infof("Downloaded %d bytes", n)

	// FIXME:  we should move that out of package git
	destPath, closeSource, err := git.TempDirAsync(ctx, tracing.NewNoop(), "s3-artifact-paths")
	if err != nil {
		return "", "", nil, errors.WithMessage(err, "get temp dir")
	}
	logger.Infof("Resources dest: %s", destPath)

	files, err := unzipFile(zipDestPath, destPath)
	if err != nil {
		// manually close destination directory here as we must allow callers to
		// access the directory and this cannot use defer
		closeSource(ctx)
		return "", "", nil, errors.WithMessage(err, "unzip file")
	}
	logger.Infof("Found files: %v", files)

	return getArtifactFilePath(files, "artifact.json"), path.Join(destPath, environment), closeSource, nil
}

func getArtifactFilePath(files []string, fileName string) string {
	for _, file := range files {
		if strings.HasSuffix(file, fileName) {
			return file
		}
	}
	return ""
}

func (f *Service) LatestArtifactSpecification(ctx context.Context, service string, branch string) (artifact.Spec, error) {
	key, err := f.getLatestObjectKey(ctx, service, branch)

	if err != nil {
		return artifact.Spec{}, err
	}

	return f.getArtifactSpecFromObjectKey(ctx, getObjectKeyName(service, key))
}

func (f *Service) LatestArtifactPaths(ctx context.Context, service string, environment string, branch string) (specPath string, resourcesPath string, close func(context.Context), err error) {
	return "", "", nil, fmt.Errorf("artifact not found")
}

func (f *Service) ArtifactSpecifications(ctx context.Context, service string, n int) ([]artifact.Spec, error) {
	list, err := f.s3client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:  aws.String(f.bucketName),
		MaxKeys: aws.Int64(1000), // TODO: Find a solution to handle more than 1000
		Prefix:  aws.String(getServiceObjectKeyPrefix(service)),
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(list.Contents, func(i, j int) bool {
		return list.Contents[i].LastModified.After(*list.Contents[j].LastModified)
	})

	var artifactSpecs []artifact.Spec
	for _, object := range list.Contents {
		artifactSpec, err := f.getArtifactSpecFromObjectKey(ctx, *object.Key)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed getting object %s", *object.Key)
		}
		artifactSpecs = append(artifactSpecs, artifactSpec)
		if len(artifactSpecs) >= n {
			break
		}
	}

	return artifactSpecs, nil
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

func (f *Service) getArtifactSpecFromObjectKey(ctx context.Context, objectKey string) (artifact.Spec, error) {
	head, err := f.s3client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(f.bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return artifact.Spec{}, err
	}

	artifactSpec, err := decodeSpecFromMetadata(head.Metadata)

	if err != nil {
		return artifact.Spec{}, err
	}

	return artifactSpec, nil
}
