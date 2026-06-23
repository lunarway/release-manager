package metrics

import (
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// newTestObserver builds an Observer whose flowDuration histogram is registered
// against a fresh, isolated registry so tests can run in parallel without
// triggering duplicate-registration panics.
func newTestObserver(t *testing.T) (*Observer, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	hist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "release_manager_flow_duration_seconds",
		Help:    "Duration of flow operations in seconds",
		Buckets: []float64{.05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60, 120, 300},
	}, []string{"operation", "outcome"})
	reg.MustRegister(hist)
	return &Observer{flowDuration: hist}, reg
}

// newTestObserverForPushDuration builds an Observer whose releasePushDuration
// histogram is registered against a fresh, isolated registry.
func newTestObserverForPushDuration(t *testing.T) (*Observer, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()
	hist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "release_manager_release_push_duration_seconds",
		Help:    "Wall-clock duration from release intent accepted to release pushed to GitHub, in seconds",
		Buckets: []float64{.25, .5, 1, 2.5, 5, 10, 20, 30, 60, 120, 180, 300, 600},
	}, []string{"outcome"})
	reg.MustRegister(hist)
	return &Observer{releasePushDuration: hist}, reg
}

// TestObserveReleasePushDuration verifies that ObserveReleasePushDuration maps
// nil errors to the "success" outcome and non-nil errors to the "error" outcome,
// and that exactly one sample is recorded per call.
func TestObserveReleasePushDuration(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		err             error
		expectedOutcome string
	}{
		{
			name:            "nil error yields success outcome",
			err:             nil,
			expectedOutcome: "success",
		},
		{
			name:            "non-nil error yields error outcome",
			err:             errors.New("push failed"),
			expectedOutcome: "error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			obs, reg := newTestObserverForPushDuration(t)
			obs.ObserveReleasePushDuration(time.Now(), tc.err)

			mfs, err := reg.Gather()
			if err != nil {
				t.Fatalf("gather metrics: %v", err)
			}

			var found bool
			for _, mf := range mfs {
				if mf.GetName() != "release_manager_release_push_duration_seconds" {
					continue
				}
				for _, m := range mf.GetMetric() {
					var outcome string
					for _, lp := range m.GetLabel() {
						if lp.GetName() == "outcome" {
							outcome = lp.GetValue()
						}
					}
					if outcome == tc.expectedOutcome {
						got := m.GetHistogram().GetSampleCount()
						if got != 1 {
							t.Errorf("expected sample count 1, got %d", got)
						}
						found = true
					}
				}
			}
			if !found {
				t.Errorf("no metric found for outcome=%q", tc.expectedOutcome)
			}
		})
	}
}

// TestObserveFlowDuration verifies that ObserveFlowDuration maps nil errors to
// the "success" outcome and non-nil errors to the "error" outcome, and that
// exactly one sample is recorded per call.
func TestObserveFlowDuration(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		operation       string
		err             error
		expectedOutcome string
	}{
		{
			name:            "nil error yields success outcome",
			operation:       "deploy",
			err:             nil,
			expectedOutcome: "success",
		},
		{
			name:            "non-nil error yields error outcome",
			operation:       "rollback",
			err:             errors.New("something went wrong"),
			expectedOutcome: "error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			obs, reg := newTestObserver(t)
			obs.ObserveFlowDuration(tc.operation, time.Now(), tc.err)

			mfs, err := reg.Gather()
			if err != nil {
				t.Fatalf("gather metrics: %v", err)
			}

			var found bool
			for _, mf := range mfs {
				if mf.GetName() != "release_manager_flow_duration_seconds" {
					continue
				}
				for _, m := range mf.GetMetric() {
					var op, outcome string
					for _, lp := range m.GetLabel() {
						switch lp.GetName() {
						case "operation":
							op = lp.GetValue()
						case "outcome":
							outcome = lp.GetValue()
						}
					}
					if op == tc.operation && outcome == tc.expectedOutcome {
						got := m.GetHistogram().GetSampleCount()
						if got != 1 {
							t.Errorf("expected sample count 1, got %d", got)
						}
						found = true
					}
				}
			}
			if !found {
				t.Errorf("no metric found for operation=%q outcome=%q", tc.operation, tc.expectedOutcome)
			}
		})
	}
}
