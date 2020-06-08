package kubernetes

import (
	"context"
	"net/http"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"go.uber.org/multierr"
)

// Exporter sends a formatted event to an upstream.
type Exporter interface {
	// Send a message through the exporter.
	SendSuccessfulReleaseEvent(c context.Context, event httpinternal.ReleaseEvent) error
	SendPodErrorEvent(c context.Context, event httpinternal.PodErrorEvent) error
}

type ReleaseManagerExporter struct {
	Log         *log.Logger
	Environment string
	Clients     []httpinternal.Client
}

func (e *ReleaseManagerExporter) SendSuccessfulReleaseEvent(ctx context.Context, event httpinternal.ReleaseEvent) error {
	e.Log.With("event", event).Infof("SuccesfulRelease Event")
	event.Environment = e.Environment
	return e.notifyServers(ctx, event)
}

func (e *ReleaseManagerExporter) SendPodErrorEvent(ctx context.Context, event httpinternal.PodErrorEvent) error {
	e.Log.With("event", event).Infof("PodError Event")
	event.Environment = e.Environment
	return e.notifyServers(ctx, event)
}

func (e *ReleaseManagerExporter) notifyServers(ctx context.Context, event interface{}) error {
	var errs error
	for _, client := range e.Clients {
		var resp httpinternal.KubernetesNotifyResponse
		url, err := client.URL("webhook/daemon/k8s/deploy")
		if err != nil {
			errs = multierr.Append(errs, err)
			continue
		}
		err = client.Do(http.MethodPost, url, event, &resp)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	return errs
}
