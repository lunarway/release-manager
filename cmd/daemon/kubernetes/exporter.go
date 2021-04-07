package kubernetes

import (
	"context"
	"net/http"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
)

// Exporter sends a formatted event to an upstream.
type Exporter interface {
	// Send a message through the exporter.
	SendSuccessfulReleaseEvent(c context.Context, event httpinternal.ReleaseEvent) error
	SendPodErrorEvent(c context.Context, event httpinternal.PodErrorEvent) error
	SendJobErrorEvent(c context.Context, event httpinternal.JobErrorEvent) error
}

type ReleaseManagerExporter struct {
	Log         *log.Logger
	Environment string
	Client      httpinternal.Client
}

func (e *ReleaseManagerExporter) SendSuccessfulReleaseEvent(ctx context.Context, event httpinternal.ReleaseEvent) error {
	e.Log.With("event", event).Infof("SuccesfulRelease Event")
	var resp httpinternal.KubernetesNotifyResponse
	url, err := e.Client.URL("webhook/daemon/k8s/deploy")
	if err != nil {
		return err
	}
	event.Environment = e.Environment
	err = e.Client.Do(http.MethodPost, url, event, &resp)
	if err != nil {
		return err
	}
	return nil
}

func (e *ReleaseManagerExporter) SendPodErrorEvent(ctx context.Context, event httpinternal.PodErrorEvent) error {
	e.Log.With("event", event).Infof("PodError Event")
	var resp httpinternal.KubernetesNotifyResponse
	url, err := e.Client.URL("webhook/daemon/k8s/error")
	if err != nil {
		return err
	}
	event.Environment = e.Environment
	err = e.Client.Do(http.MethodPost, url, event, &resp)
	if err != nil {
		return err
	}
	return nil
}

func (e *ReleaseManagerExporter) SendJobErrorEvent(ctx context.Context, event httpinternal.JobErrorEvent) error {
	e.Log.With("event", event).Infof("JobError Event")
	var resp httpinternal.KubernetesNotifyResponse
	url, err := e.Client.URL("webhook/daemon/k8s/joberror")
	if err != nil {
		return err
	}
	event.Environment = e.Environment
	err = e.Client.Do(http.MethodPost, url, event, &resp)
	if err != nil {
		return err
	}
	return nil
}
