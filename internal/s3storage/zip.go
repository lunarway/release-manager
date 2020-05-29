package s3storage

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

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
