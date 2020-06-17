package flow

import (
	"context"

	"github.com/lunarway/release-manager/internal/artifact"
)

// DescribeArtifact returns n artifacts for a service.
func (s *Service) DescribeArtifact(ctx context.Context, service string, n int) ([]artifact.Spec, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.DescribeArtifact")
	defer span.Finish()
	return s.Storage.ArtifactSpecifications(ctx, service, n)
}

// DescribeArtifact returns n artifacts for a service.
func (s *Service) DescribeLatestArtifact(ctx context.Context, service, branch string) (artifact.Spec, error) {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.DescribeLatestArtifact")
	defer span.Finish()
	return s.Storage.LatestArtifactSpecification(ctx, service, branch)
}
