package fallbackstorage

import (
	"context"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/log"
)

// TODO: Describe how it works like only looking in primary when primary works
type Fallback struct {
	primary   flow.ArtifactReadStorage
	secondary flow.ArtifactReadStorage
}

func New(primary flow.ArtifactReadStorage, secondary flow.ArtifactReadStorage) *Fallback {
	return &Fallback{
		primary:   primary,
		secondary: secondary,
	}
}

func (f *Fallback) ArtifactExists(ctx context.Context, artifactID string) (bool, error) {
	exists, primaryErr := f.primary.ArtifactExists(ctx, artifactID)
	if primaryErr != nil {
		log.WithContext(ctx).WithFields("storageType", "fallback").Infof("storage: fallback: ArtifactExists failed for primary: %s", primaryErr)
		return f.secondary.ArtifactExists(ctx, artifactID)
	}
	if !exists {
		return f.secondary.ArtifactExists(ctx, artifactID)
	}
	return true, nil
}

func (f *Fallback) ArtifactSpecification(ctx context.Context, service string, artifactID string) (artifact.Spec, error) {
	primary, primaryErr := f.primary.ArtifactSpecification(ctx, service, artifactID)
	if primaryErr != nil {
		log.WithContext(ctx).WithFields("storageType", "fallback").Infof("storage: fallback: ArtifactSpecification failed for primary: %s", primaryErr)
		return f.secondary.ArtifactSpecification(ctx, service, artifactID)
	}
	return primary, nil
}

func (f *Fallback) ArtifactPaths(ctx context.Context, service string, environment string, branch string, artifactID string) (specPath string, resourcesPath string, close func(context.Context), err error) {
	specPath, resourcePath, close, err := f.primary.ArtifactPaths(ctx, service, environment, branch, artifactID)
	if err != nil {
		log.WithContext(ctx).WithFields("storageType", "fallback").Infof("storage: fallback: ArtifactPaths failed for primary: %s", err)
		return f.secondary.ArtifactPaths(ctx, service, environment, branch, artifactID)
	}
	return specPath, resourcePath, close, nil
}

func (f *Fallback) LatestArtifactSpecification(ctx context.Context, service string, branch string) (artifact.Spec, error) {
	primary, primaryErr := f.primary.LatestArtifactSpecification(ctx, service, branch)
	if primaryErr != nil {
		log.WithContext(ctx).WithFields("storageType", "fallback").Infof("storage: fallback: ArtifactSpecification failed for primary: %s", primaryErr)
		return f.secondary.LatestArtifactSpecification(ctx, service, branch)
	}
	return primary, nil
}

func (f *Fallback) LatestArtifactPaths(ctx context.Context, service string, environment string, branch string) (specPath string, resourcesPath string, close func(context.Context), err error) {
	specPath, resourcePath, close, err := f.primary.LatestArtifactPaths(ctx, service, environment, branch)
	if err != nil {
		log.WithContext(ctx).WithFields("storageType", "fallback").Infof("storage: fallback: ArtifactPaths failed for primary: %s", err)
		return f.secondary.LatestArtifactPaths(ctx, service, environment, branch)
	}
	return specPath, resourcePath, close, nil
}

// ArtifactSpecifications takes as many as can be found in primary and the rest from secondary
func (f *Fallback) ArtifactSpecifications(ctx context.Context, service string, n int) ([]artifact.Spec, error) {
	panic("not implemented") // TODO: Implement
}
