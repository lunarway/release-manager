package s3storage

import (
	"context"
	"fmt"

	"github.com/lunarway/release-manager/internal/artifact"
)

// ArtifactExists returns whether an artifact with id artifactID is available.
func (f *Service) ArtifactExists(ctx context.Context, artifactID string) (bool, error) {
	return false, nil
}

// ArtifactSpecification returns the artifact specification for a given
// service and artifact ID.
func (f *Service) ArtifactSpecification(ctx context.Context, service string, artifactID string) (artifact.Spec, error) {
	return artifact.Spec{}, fmt.Errorf("artifact not found")
}

// ArtifactPaths returns file system paths for the artifact specification
// (specPath) and yaml resources directory (resourcesPath) available on the
// file system for copying to releases. The returned close function is
// responsible for clean up of the persisted files.
func (f *Service) ArtifactPaths(ctx context.Context, service string, environment string, branch string, artifactID string) (specPath string, resourcesPath string, close func(context.Context), err error) {
	return "", "", nil, fmt.Errorf("artifact not found")
}

// LatestArtifactSpecification returns the latest artifact specification for a
// given service and branch.
func (f *Service) LatestArtifactSpecification(ctx context.Context, service string, branch string) (artifact.Spec, error) {
	return artifact.Spec{}, fmt.Errorf("artifact not found")
}

// LatestArtifactPaths returns file system paths for the artifact
// specification (specPath) and yaml resources directory (resourcesPath)
// available on the file system for copying to releases of the latest artifact
// for provided service and branch. The returned close function is responsible
// for clean up of the persisted files.
func (f *Service) LatestArtifactPaths(ctx context.Context, service string, environment string, branch string) (specPath string, resourcesPath string, close func(context.Context), err error) {
	return "", "", nil, fmt.Errorf("artifact not found")
}

// ArtifactSpecifications returns a list of n newest artifact specifications
// for service. They should be ordered by newest first.
func (f *Service) ArtifactSpecifications(ctx context.Context, service string, n int) ([]artifact.Spec, error) {
	return nil, nil
}
