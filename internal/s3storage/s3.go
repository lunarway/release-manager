package s3storage

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

// downloadArtifact downloads an artifact from its AWS S3 key. The returned
// string is a file system path to a raw artifact, ie. unzipped.
func (f *Service) downloadArtifact(ctx context.Context, key string) (string, func(context.Context), error) {
	span, ctx := f.tracer.FromCtx(ctx, "s3storage.downloadArtifact")
	defer span.Finish()

	logger := log.WithContext(ctx)
	logger.WithFields("key", key).Infof("Downloading artifact from S3 key '%s'", key)
	zipDest, err := os.CreateTemp("", "s3-artifact-zip")
	if err != nil {
		return "", nil, errors.WithMessage(err, "create temp file for zip")
	}

	logger.Debugf("Zip destination: %s", zipDest.Name())

	defer func() {
		err := zipDest.Close()
		if err != nil {
			logger.Errorf("Failed to close zip destination file: %v", err)
		}
		err = os.Remove(zipDest.Name())
		if err != nil {
			logger.Errorf("Failed to remove zip destination file '%s': %v", zipDest.Name(), err)
		}
	}()

	downloader := s3manager.NewDownloaderWithClient(f.s3client)
	n, err := downloader.DownloadWithContext(ctx, zipDest, &s3.GetObjectInput{
		Bucket: aws.String(f.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NoSuchKey" {
			return "", nil, flow.ErrArtifactNotFound
		}
		return "", nil, errors.WithMessage(err, "download object")
	}
	logger.Infof("Downloaded %d bytes", n)

	destPath, closeSource, err := tempDir("s3-artifact-paths")
	if err != nil {
		return "", nil, errors.WithMessage(err, "get temp dir")
	}
	logger.Infof("Artifact destination: %s", destPath)

	span, _ = f.tracer.FromCtx(ctx, "unzip artifact")
	files, err := unzipFile(zipDest.Name(), destPath)
	defer span.Finish()
	if err != nil {
		// manually close destination directory here as we must allow callers to
		// access the directory and this cannot use defer
		closeSource(ctx)
		return "", nil, errors.WithMessage(err, "unzip file")
	}
	logger.WithFields("files", files).Infof("Artifact contains %d files", len(files))
	return destPath, closeSource, nil
}

func tempDir(prefix string) (string, func(context.Context), error) {
	path, err := os.MkdirTemp("", prefix)
	if err != nil {
		return "", func(context.Context) {}, err
	}
	return path, func(ctx context.Context) {
		err := os.RemoveAll(path)
		if err != nil {
			log.WithContext(ctx).Errorf("Removing temporary directory failed: path '%s': %v", path, err)
		}
	}, nil
}

func unzipFile(src, destination string) (filenames []string, err error) {
	var r *zip.ReadCloser
	r, err = zip.OpenReader(src)
	if err != nil {
		err = errors.WithMessagef(err, "open zip '%s'", src)
		return
	}
	defer checkClose(r, &err, "zip reader")

	for _, f := range r.File {
		var rc io.ReadCloser
		rc, err = f.Open()
		if err != nil {
			err = errors.WithMessagef(err, "open file '%s'", f.Name)
			return
		}
		defer checkClose(rc, &err, "source file")

		var fpath string
		fpath, err = securejoin.SecureJoin(destination, f.Name)
		if err != nil {
			err = errors.WithMessagef(err, "join destination path for file '%s'", f.Name)
			return
		}
		if f.FileInfo().IsDir() {
			err = os.MkdirAll(fpath, os.ModePerm)
			if err != nil {
				err = errors.WithMessagef(err, "create directory '%s'", fpath)
				return
			}
			continue
		}
		if lastIndex := strings.LastIndex(fpath, string(os.PathSeparator)); lastIndex > -1 {
			fdir := fpath[:lastIndex]
			err = os.MkdirAll(fdir, os.ModePerm)
			if err != nil {
				err = errors.WithMessagef(err, "create directory '%s'", fdir)
				return
			}
		}
		var file *os.File
		file, err = os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			err = errors.WithMessagef(err, "open file '%s'", fpath)
			return
		}
		defer checkClose(file, &err, "destination file")

		_, err = io.Copy(file, rc)
		if err != nil {
			err = errors.WithMessagef(err, "copy zip file '%s' to file '%s'", f.Name, fpath)
			return
		}
		filenames = append(filenames, fpath)
	}
	return filenames, nil
}

func checkClose(c io.Closer, err *error, action string) {
	cerr := c.Close()
	if cerr != nil {
		if err == nil {
			err = &cerr
			return
		}
		log.Errorf("unzipper: %s: close failed: %v", action, cerr)
	}
}
