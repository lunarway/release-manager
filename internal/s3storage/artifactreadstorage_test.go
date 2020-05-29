package s3storage_test

import (
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/s3storage"
)

var _ flow.ArtifactReadStorage = &s3storage.Service{}
