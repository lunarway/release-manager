package fallbackstorage_test

import (
	"context"
	"testing"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/fallbackstorage"
	"github.com/lunarway/release-manager/internal/flow"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

var _ flow.ArtifactReadStorage = &fallbackstorage.Fallback{}

func TestFallback_ArtifactSpecifications(t *testing.T) {
	spec := func(id string) artifact.Spec {
		return artifact.Spec{
			ID: id,
		}
	}
	tt := []struct {
		name           string
		n              int
		primarySpecs   []artifact.Spec
		secondarySpecs []artifact.Spec
		output         []artifact.Spec
	}{
		{
			name: "all in primary",
			n:    2,
			primarySpecs: []artifact.Spec{
				spec("1"),
				spec("2"),
			},
			secondarySpecs: nil,
			output: []artifact.Spec{
				spec("1"),
				spec("2"),
			},
		},
		{
			name:         "all in secondary",
			n:            2,
			primarySpecs: nil,
			secondarySpecs: []artifact.Spec{
				spec("1"),
				spec("2"),
			},
			output: []artifact.Spec{
				spec("1"),
				spec("2"),
			},
		},
		{
			name: "subset in primary",
			n:    2,
			primarySpecs: []artifact.Spec{
				spec("1"),
			},
			secondarySpecs: []artifact.Spec{
				spec("2"),
				spec("3"),
			},
			output: []artifact.Spec{
				spec("1"),
				spec("2"),
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			log.Init(&log.Configuration{
				Level: log.Level{
					Level: zapcore.DebugLevel,
				},
				Development: true,
			})
			primary := mockStorage{
				specs: tc.primarySpecs,
			}
			secondary := mockStorage{
				specs: tc.secondarySpecs,
			}
			storage := fallbackstorage.New(&primary, &secondary, tracing.NewNoop())

			specs, err := storage.ArtifactSpecifications(context.Background(), "", tc.n)
			if !assert.NoError(t, err, "unexpected error when getting specs") {
				return
			}
			assert.Equal(t, tc.output, specs, "specs not as expected")
		})
	}
}

type mockStorage struct {
	specs []artifact.Spec
}

// ArtifactSpecifications returns a list of n newest artifact specifications
// for service. They should be ordered by newest first.
func (m *mockStorage) ArtifactSpecifications(ctx context.Context, service string, n int) ([]artifact.Spec, error) {
	var output []artifact.Spec
	for _, s := range m.specs {
		output = append(output, s)
		if len(output) == n {
			return output, nil
		}
	}
	return output, nil
}

// ArtifactExists returns whether an artifact with id artifactID is available.
func (m *mockStorage) ArtifactExists(ctx context.Context, service string, artifactID string) (bool, error) {
	panic("not implemented") // TODO: Implement
}

// ArtifactSpecification returns the artifact specification for a given
// service and artifact ID.
func (m *mockStorage) ArtifactSpecification(ctx context.Context, service string, artifactID string) (artifact.Spec, error) {
	panic("not implemented") // TODO: Implement
}

// ArtifactPaths returns file system paths for the artifact specification
// (specPath) and yaml resources directory (resourcesPath) available on the
// file system for copying to releases. The returned close function is
// responsible for clean up of the persisted files.
func (m *mockStorage) ArtifactPaths(ctx context.Context, service string, environment string, branch string, artifactID string) (specPath string, resourcesPath string, close func(context.Context), err error) {
	panic("not implemented") // TODO: Implement
}

// LatestArtifactSpecification returns the latest artifact specification for a
// given service and branch.
func (m *mockStorage) LatestArtifactSpecification(ctx context.Context, service string, branch string) (artifact.Spec, error) {
	panic("not implemented") // TODO: Implement
}

// LatestArtifactPaths returns file system paths for the artifact
// specification (specPath) and yaml resources directory (resourcesPath)
// available on the file system for copying to releases of the latest artifact
// for provided service and branch. The returned close function is responsible
// for clean up of the persisted files.
func (m *mockStorage) LatestArtifactPaths(ctx context.Context, service string, environment string, branch string) (specPath string, resourcesPath string, close func(context.Context), err error) {
	panic("not implemented") // TODO: Implement
}
