package flow

import (
	"context"

	"github.com/lunarway/release-manager/internal/artifact"
)

type ArtifactReadStorage interface {
	// ArtifactExists returns whether an artifact with id artifactID is available.
	ArtifactExists(ctx context.Context, service, artifactID string) (bool, error)

	// ArtifactSpecification returns the artifact specification for a given
	// service and artifact ID.
	ArtifactSpecification(ctx context.Context, service, artifactID string) (artifact.Spec, error)

	// ArtifactPaths returns file system paths for the artifact specification
	// (specPath) and yaml resources directory (resourcesPath) available on the
	// file system for copying to releases. The returned close function is
	// responsible for clean up of the persisted files.
	ArtifactPaths(ctx context.Context, service, environment, branch, artifactID string) (specPath, resourcesPath string, close func(context.Context), err error)

	// LatestArtifactSpecification returns the latest artifact specification for a
	// given service and branch.
	LatestArtifactSpecification(ctx context.Context, service, branch string) (artifact.Spec, error)

	// LatestArtifactPaths returns file system paths for the artifact
	// specification (specPath) and yaml resources directory (resourcesPath)
	// available on the file system for copying to releases of the latest artifact
	// for provided service and branch. The returned close function is responsible
	// for clean up of the persisted files.
	LatestArtifactPaths(ctx context.Context, service, environment, branch string) (specPath, resourcesPath string, close func(context.Context), err error)

	// ArtifactSpecifications returns a list of n newest artifact specifications
	// for service. They should be ordered by newest first.
	ArtifactSpecifications(ctx context.Context, service string, n int) ([]artifact.Spec, error)
}
