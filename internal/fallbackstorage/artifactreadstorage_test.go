package fallbackstorage_test

import (
	"github.com/lunarway/release-manager/internal/fallbackstorage"
	"github.com/lunarway/release-manager/internal/flow"
)

var _ flow.ArtifactReadStorage = &fallbackstorage.Fallback{}
