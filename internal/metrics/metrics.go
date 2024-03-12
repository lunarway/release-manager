package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Observer struct {
	releaseCounter *prometheus.CounterVec
}

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
	}
}

type Release struct {
	Environment string
	Service     string
	Releaser    string
	Intent      string
	Squad       string
}

func (o *Observer) ObserveRelease(release Release) {
	o.releaseCounter.WithLabelValues(
		release.Environment,
		release.Service,
		release.Releaser,
		release.Intent,
		release.Squad,
	).Inc()
}
