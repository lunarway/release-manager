package flow

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lunarway/release-manager/internal/artifact"
	"github.com/lunarway/release-manager/internal/copy"
	internalgit "github.com/lunarway/release-manager/internal/git"
	"github.com/lunarway/release-manager/internal/intent"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestMain(m *testing.M) {
	log.Init(&log.Configuration{
		Level:       log.Level{Level: zapcore.ErrorLevel},
		Development: false,
	})
	os.Exit(m.Run())
}

// fakeObserver captures ObserveFlowDuration and ObserveReleasePushDuration calls.
type fakeObserver struct {
	flowCalls []flowCall
	pushCalls []pushCall
}

type flowCall struct {
	operation string
	start     time.Time
	err       error
}

type pushCall struct {
	start time.Time
	err   error
}

func (f *fakeObserver) ObserveFlowDuration(operation string, start time.Time, err error) {
	f.flowCalls = append(f.flowCalls, flowCall{operation: operation, start: start, err: err})
}

func (f *fakeObserver) ObserveReleasePushDuration(start time.Time, err error) {
	f.pushCalls = append(f.pushCalls, pushCall{start: start, err: err})
}

// fakeStorage is a minimal ArtifactReadStorage backed by files on disk.
type fakeStorage struct {
	specPath      string
	resourcesPath string
}

func (f *fakeStorage) ArtifactExists(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

func (f *fakeStorage) ArtifactSpecification(_ context.Context, _, _ string) (artifact.Spec, error) {
	return artifact.Spec{}, nil
}

func (f *fakeStorage) ArtifactPaths(_ context.Context, _, _, _, _ string) (string, string, func(context.Context), error) {
	return f.specPath, f.resourcesPath, func(context.Context) {}, nil
}

func (f *fakeStorage) LatestArtifactPaths(_ context.Context, _, _, _ string) (string, string, func(context.Context), error) {
	return f.specPath, f.resourcesPath, func(context.Context) {}, nil
}

func (f *fakeStorage) LatestArtifactSpecification(_ context.Context, _, _ string) (artifact.Spec, error) {
	return artifact.Spec{}, nil
}

func (f *fakeStorage) ArtifactSpecifications(_ context.Context, _ string, _ int, _ string) ([]artifact.Spec, error) {
	return nil, nil
}

// setupArtifactStorage creates a temp dir with a minimal artifact.json.
func setupArtifactStorage(t *testing.T) *fakeStorage {
	t.Helper()
	dir := t.TempDir()
	specPath := filepath.Join(dir, "artifact.json")
	spec := artifact.Spec{
		ID:      "master-test-1234",
		Service: "svc",
		Application: artifact.Repository{
			Branch:      "master",
			AuthorName:  "test",
			AuthorEmail: "test@example.com",
		},
	}
	b, err := json.Marshal(spec)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(specPath, b, 0600))
	return &fakeStorage{
		specPath:      specPath,
		resourcesPath: dir,
	}
}

// newTestService builds a Service for ExecReleaseArtifactID unit tests.
func newTestService(t *testing.T, obs FlowObserver, gitSvc GitService, storage ArtifactReadStorage) *Service {
	t.Helper()
	logger := log.New(&log.Configuration{
		Level:       log.Level{Level: zapcore.ErrorLevel},
		Development: false,
	})
	return &Service{
		ArtifactFileName: "artifact.json",
		Git:              gitSvc,
		Tracer:           tracing.NewNoop(),
		Storage:          storage,
		Copier:           copy.New(logger),
		Observer:         obs,
		MaxRetries:       1,
	}
}

// TestReleaseArtifactIDEvent_EnqueuedAt_roundtrip verifies that EnqueuedAt
// survives Marshal → Unmarshal intact.
func TestReleaseArtifactIDEvent_EnqueuedAt_roundtrip(t *testing.T) {
	t.Parallel()

	// JSON time is RFC3339 — truncate to second precision so round-trip is exact.
	enqueuedAt := time.Now().UTC().Truncate(time.Second)
	original := ReleaseArtifactIDEvent{
		Service:     "svc",
		Environment: "dev",
		ArtifactID:  "master-abc-123",
		EnqueuedAt:  enqueuedAt,
	}

	data, err := original.Marshal()
	require.NoError(t, err)

	var got ReleaseArtifactIDEvent
	require.NoError(t, got.Unmarshal(data))

	assert.True(t, original.EnqueuedAt.Equal(got.EnqueuedAt),
		"EnqueuedAt not preserved through Marshal/Unmarshal: want %v got %v",
		original.EnqueuedAt, got.EnqueuedAt)
	assert.Equal(t, original.Service, got.Service)
	assert.Equal(t, original.ArtifactID, got.ArtifactID)
}

// TestExecReleaseArtifactID_observationWiring tests that ObserveReleasePushDuration
// is called correctly under different outcome scenarios.
func TestExecReleaseArtifactID_observationWiring(t *testing.T) {
	t.Parallel()

	enqueuedAt := time.Now().Add(-5 * time.Second)
	terminalErr := errors.New("commit failed")

	cases := []struct {
		name              string
		enqueuedAt        time.Time
		commitErr         error
		wantPushCallCount int
		wantPushErrNil    bool
	}{
		{
			name:              "success: push succeeds",
			enqueuedAt:        enqueuedAt,
			commitErr:         nil,
			wantPushCallCount: 1,
			wantPushErrNil:    true,
		},
		{
			name:              "terminal error: ObserveReleasePushDuration called with non-nil err",
			enqueuedAt:        enqueuedAt,
			commitErr:         terminalErr,
			wantPushCallCount: 1,
			wantPushErrNil:    false,
		},
		{
			name:              "no-op (ErrNothingToCommit): ObserveReleasePushDuration NOT called",
			enqueuedAt:        enqueuedAt,
			commitErr:         internalgit.ErrNothingToCommit,
			wantPushCallCount: 0,
		},
		{
			name:              "zero EnqueuedAt (legacy message): ObserveReleasePushDuration NOT called",
			enqueuedAt:        time.Time{},
			commitErr:         nil,
			wantPushCallCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			obs := &fakeObserver{}
			storage := setupArtifactStorage(t)

			gitSvc := &MockGitService{}
			gitSvc.Test(t)
			// ShallowClone is called with a temp dir — match any destination.
			gitSvc.On("ShallowClone", mock.Anything, mock.AnythingOfType("string")).Return(nil)
			gitSvc.On("Commit", mock.Anything, mock.AnythingOfType("string"), ".", mock.AnythingOfType("string")).Return(tc.commitErr)

			svc := newTestService(t, obs, gitSvc, storage)

			event := ReleaseArtifactIDEvent{
				Service:     "svc",
				Environment: "dev",
				Namespace:   "dev",
				ArtifactID:  "master-test-1234",
				Branch:      "master",
				Intent:      intent.NewReleaseArtifact(),
				EnqueuedAt:  tc.enqueuedAt,
			}

			_ = svc.ExecReleaseArtifactID(context.Background(), event)

			assert.Len(t, obs.pushCalls, tc.wantPushCallCount, "ObserveReleasePushDuration call count")
			if tc.wantPushCallCount > 0 {
				call := obs.pushCalls[0]
				assert.True(t, tc.enqueuedAt.Equal(call.start),
					"start time: want %v got %v", tc.enqueuedAt, call.start)
				if tc.wantPushErrNil {
					assert.NoError(t, call.err)
				} else {
					assert.Error(t, call.err)
				}
			}
			// ObserveFlowDuration must always be called exactly once.
			assert.Len(t, obs.flowCalls, 1, "ObserveFlowDuration must always be called once")
		})
	}
}
