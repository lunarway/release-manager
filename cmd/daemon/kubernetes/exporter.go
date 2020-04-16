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
	SendSuccessfulDeploymentEvent(c context.Context, event httpinternal.DeploymentEvent) error
	SendPodErrorEvent(c context.Context, event httpinternal.PodErrorEvent) error
}

type ReleaseManagerExporter struct {
	Log         *log.Logger
	Environment string
	Client      httpinternal.Client
}

func (e *ReleaseManagerExporter) SendSuccessfulDeploymentEvent(ctx context.Context, event httpinternal.DeploymentEvent) error {
	e.Log.With("event", event).Infof("SuccesfulDeployment Event")
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

func (e *ReleaseManagerExporter) SendPodErrorEvent(c context.Context, event httpinternal.PodErrorEvent) error {
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
