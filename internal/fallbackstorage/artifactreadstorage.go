package fallbackstorage

import (
	"context"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/tracing"
)

// Fallback is a flow.ArtifactReadStorage implementation that combines the
// storage of two implementations, a primary and secondary. If the primary
// storage returns an error the secondary storage is invoked.
type Fallback struct {
	primary   flow.ArtifactReadStorage
	secondary flow.ArtifactReadStorage
	tracer    tracing.Tracer
}

func New(primary flow.ArtifactReadStorage, secondary flow.ArtifactReadStorage, tracer tracing.Tracer) *Fallback {
	return &Fallback{
		primary:   primary,
		secondary: secondary,
		tracer:    tracer,
	}
}

func (f *Fallback) ArtifactExists(ctx context.Context, service, artifactID string) (bool, error) {
	span, primaryCtx := f.tracer.FromCtx(ctx, "fallback.ArtifactExists primary")

	exists, primaryErr := f.primary.ArtifactExists(primaryCtx, service, artifactID)

	span.Finish()

	if primaryErr != nil {
		log.WithContext(ctx).WithFields("storageType", "fallback").Infof("storage: fallback: ArtifactExists failed for primary: %s", primaryErr)

		span, ctx = f.tracer.FromCtx(ctx, "fallback.ArtifactExists secondary")
		defer span.Finish()

		return f.secondary.ArtifactExists(ctx, service, artifactID)
	}
	if !exists {
		span, _ = f.tracer.FromCtx(ctx, "fallback.ArtifactExists secondary")
		defer span.Finish()

		return f.secondary.ArtifactExists(ctx, service, artifactID)
	}
	return true, nil
}

func (f *Fallback) ArtifactSpecification(ctx context.Context, service string, artifactID string) (artifact.Spec, error) {
	span, primaryCtx := f.tracer.FromCtx(ctx, "fallback.ArtifactSpecification primary")

	primary, primaryErr := f.primary.ArtifactSpecification(primaryCtx, service, artifactID)

	span.Finish()

	if primaryErr != nil {
		log.WithContext(ctx).WithFields("storageType", "fallback").Infof("storage: fallback: ArtifactSpecification failed for primary: %s", primaryErr)

		span, ctx := f.tracer.FromCtx(ctx, "fallback.ArtifactSpecification secondary")
		defer span.Finish()

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
		log.WithContext(ctx).WithFields("storageType", "fallback").Infof("storage: fallback: LatestArtifactSpecification failed for primary: %s", primaryErr)
		return f.secondary.LatestArtifactSpecification(ctx, service, branch)
	}
	return primary, nil
}

func (f *Fallback) LatestArtifactPaths(ctx context.Context, service string, environment string, branch string) (specPath string, resourcesPath string, close func(context.Context), err error) {
	span, primaryCtx := f.tracer.FromCtx(ctx, "fallback.LatestArtifactPaths primary")

	specPath, resourcePath, close, err := f.primary.LatestArtifactPaths(primaryCtx, service, environment, branch)

	span.Finish()

	if err != nil {
		log.WithContext(ctx).WithFields("storageType", "fallback").Infof("storage: fallback: LatestArtifactPaths failed for primary: %s", err)

		span, ctx := f.tracer.FromCtx(ctx, "fallback.LatestArtifactPaths secondary")
		defer span.Finish()

		return f.secondary.LatestArtifactPaths(ctx, service, environment, branch)
	}
	return specPath, resourcePath, close, nil
}

// ArtifactSpecifications takes as many as can be found in primary and the rest from secondary
func (f *Fallback) ArtifactSpecifications(ctx context.Context, service string, n int) ([]artifact.Spec, error) {
	span, primaryCtx := f.tracer.FromCtx(ctx, "fallback.ArtifactSpecifications primary")

	primarySpecs, err := f.primary.ArtifactSpecifications(primaryCtx, service, n)

	span.Finish()

	if err != nil {
		log.WithContext(ctx).WithFields("storageType", "fallback").Infof("storage: fallback: ArtifactSpecifications failed for primary: %s", err)
	}
	if len(primarySpecs) >= n {
		return primarySpecs, nil
	}
	log.WithContext(ctx).WithFields("storageType", "fallback").Infof("storage: fallback: ArtifactSpecifications found %d of %d in primary. Looking for the rest in secondary", len(primarySpecs), n)
	n = n - len(primarySpecs)

	span, ctx = f.tracer.FromCtx(ctx, "fallback.ArtifactSpecifications secondary")
	defer span.Finish()

	secondarySpecs, err := f.secondary.ArtifactSpecifications(ctx, service, n)
	if err != nil {
		return nil, err
	}
	return append(primarySpecs, secondarySpecs...), nil
}
