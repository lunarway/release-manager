package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Observer holds Prometheus metrics for the release manager.
type Observer struct {
	releaseCounter *prometheus.CounterVec
	flowDuration   *prometheus.HistogramVec
}

// NewObserver creates and registers all Prometheus metrics.
func NewObserver() *Observer {
	return &Observer{
		releaseCounter: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "release_manager_releases_total",
			Help: "Total number of releases",
		}, []string{
			"environment",
			"service",
			"releaser",
			"intent",
			"squad",
		}),
		flowDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "release_manager_flow_duration_seconds",
			Help:    "Duration of flow operations in seconds",
			Buckets: []float64{.05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
		}, []string{"operation", "outcome"}),
	}
}

// Release holds the metadata for a single release event.
type Release struct {
	Environment string
	Service     string
	Releaser    string
	Intent      string
	Squad       string
}

// ObserveRelease increments the release counter for the given release metadata.
func (o *Observer) ObserveRelease(release Release) {
	o.releaseCounter.WithLabelValues(
		release.Environment,
		release.Service,
		release.Releaser,
		release.Intent,
		release.Squad,
	).Inc()
}

// ObserveFlowDuration records the elapsed time since start for a flow
// operation, deriving the outcome label from err (success on nil, error
// otherwise).
func (o *Observer) ObserveFlowDuration(operation string, start time.Time, err error) {
	outcome := "success"
	if err != nil {
		outcome = "error"
	}
	o.flowDuration.WithLabelValues(operation, outcome).Observe(time.Since(start).Seconds())
}
