package s3storage

import (
	"archive/zip"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

// downloadArtifact downloads an artifact from its AWS S3 key. The returned
// string is a file system path to a raw artifact, ie. unzipped.
func (f *Service) downloadArtifact(ctx context.Context, key string) (string, func(context.Context), error) {
	logger := log.WithContext(ctx)

	zipDestPath, closeSource, err := tempDir("s3-artifact-paths")
	if err != nil {
		return "", nil, errors.WithMessage(err, "get temp dir")
	}
	defer closeSource(ctx)
	zipDestPath = path.Join(zipDestPath, "artifact.zip")
	logger.Debugf("Zip destination: %s", zipDestPath)
	zipDest, err := os.Create(zipDestPath)
	if err != nil {
		return "", nil, errors.WithMessage(err, "get temp dir")
	}
	defer func() {
		err := zipDest.Close()
		if err != nil {
			logger.Errorf("Failed to close zip destination file: %v", err)
		}
	}()

	downloader := s3manager.NewDownloaderWithClient(f.s3client)
	logger.Infof("Downloading object at key '%s'", key)
	n, err := downloader.DownloadWithContext(ctx, zipDest, &s3.GetObjectInput{
		Bucket: aws.String(f.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", nil, errors.WithMessage(err, "download object")
	}
	logger.Infof("Downloaded %d bytes", n)

	destPath, closeSource, err := tempDir("s3-artifact-paths")
	if err != nil {
		return "", nil, errors.WithMessage(err, "get temp dir")
	}
	logger.Infof("Resources dest: %s", destPath)

	files, err := unzipFile(zipDestPath, destPath)
	if err != nil {
		// manually close destination directory here as we must allow callers to
		// access the directory and this cannot use defer
		closeSource(ctx)
		return "", nil, errors.WithMessage(err, "unzip file")
	}
	logger.Infof("Found files: %v", files)
	return destPath, closeSource, nil
}

func tempDir(prefix string) (string, func(context.Context), error) {
	path, err := ioutil.TempDir("", prefix)
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

		fpath := filepath.Join(destination, f.Name)
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
