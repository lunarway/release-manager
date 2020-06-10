package s3storage_test

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"io/ioutil"

	"github.com/lunarway/release-manager/internal/artifact"
)

const (
	// Zipped artifact for test-service with artifact-id master-1234ds13g3-12s46g356g
	S3File_ZippedArtifact = "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"
	S3File_Empty          = "Cg=="
)

func RewriteArtifactWithSpec(base64file string, changer func(spec *artifact.Spec)) string {
	files := unzipBase64IntoFiles(base64file)
	spec, err := artifact.Decode(bytes.NewReader(files["artifact.json"]))
	if err != nil {
		panic(err)
	}
	changer(&spec)
	specBytes, err := artifact.Encode(spec, true)
	if err != nil {
		panic(err)
	}
	files["artifact.json"] = []byte(specBytes)
	return zipFilesToBase64(files)
}

func unzipBase64IntoFiles(input string) map[string][]byte {

	zippedBytes, err := base64.StdEncoding.DecodeString(input)
	r, err := zip.NewReader(bytes.NewReader(zippedBytes), int64(len(zippedBytes)))
	if err != nil {
		panic(err)
	}

	files := make(map[string][]byte)
	for _, zf := range r.File {
		reader, err := zf.Open()
		defer reader.Close()
		if err != nil {
			panic(err)
		}
		content, err := ioutil.ReadAll(reader)
		if err != nil {
			panic(err)
		}
		files[zf.Name] = content
	}

	return files
}

func zipFilesToBase64(input map[string][]byte) string {

	buf := bytes.NewBuffer([]byte{})
	zipWriter := zip.NewWriter(buf)

	for name, content := range input {
		contentWriter, err := zipWriter.Create(name)
		if err != nil {
			panic(err)
		}
		contentWriter.Write(content)
	}

	err := zipWriter.Close()
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
