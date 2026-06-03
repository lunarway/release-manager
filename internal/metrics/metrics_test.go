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
		Buckets: []float64{.05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
	}, []string{"operation", "outcome"})
	reg.MustRegister(hist)
	return &Observer{flowDuration: hist}, reg
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
