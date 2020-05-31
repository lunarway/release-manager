package s3storage

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/pkg/errors"
)

const (
	MetadataArtifactSpecPartialWriteKey = "artifact-spec"
	MetadataArtifactSpecFullKey         = "x-amz-meta-artifact-spec"

	// MetadataArtifactSpecPartialReadKey is in another format than MetadataArtifactSpecPartialWriteKey, since the AWS golang lib somehow
	// changes it when parsing headers.
	MetadataArtifactSpecPartialReadKey = "Artifact-Spec"
)

func EncodeSpecToMetadataContent(artifactSpec artifact.Spec) (string, error) {
	jsonSpec, err := artifact.Encode(artifactSpec, false)
	if err != nil {
		return "", nil
	}
	return base64.StdEncoding.EncodeToString([]byte(jsonSpec)), nil
}

func decodeSpecFromMetadata(metadata map[string]*string) (artifact.Spec, error) {
	jsonSpecBase64 := metadata[MetadataArtifactSpecPartialReadKey]
	if jsonSpecBase64 == nil {
		return artifact.Spec{}, fmt.Errorf("artifact-spec is missing in metadata")
	}
	jsonSpec, err := base64.StdEncoding.DecodeString(*jsonSpecBase64)
	if err != nil {
		return artifact.Spec{}, errors.Wrap(err, "decode base64 spec")
	}
	artifactSpec, err := artifact.Decode(strings.NewReader(string(jsonSpec)))
	if err != nil {
		return artifact.Spec{}, errors.WithMessage(err, "decode spec")
	}
	return artifactSpec, nil
}
