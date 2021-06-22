package kubernetes

import (
	"context"

	"github.com/go-openapi/runtime"
	"github.com/lunarway/release-manager/generated/http/client/webhook"
	"github.com/lunarway/release-manager/generated/http/models"
	"github.com/lunarway/release-manager/internal/log"
)

// Exporter sends a formatted event to an upstream.
type Exporter interface {
	// Send a message through the exporter.
	SendSuccessfulReleaseEvent(c context.Context, event models.DaemonKubernetesDeploymentWebhookRequest) error
	SendPodErrorEvent(c context.Context, event models.DaemonKubernetesErrorWebhookRequest) error
	SendJobErrorEvent(c context.Context, event models.DaemonKubernetesJobErrorWebhookRequest) error
}

type ReleaseManagerExporter struct {
	Log         *log.Logger
	Environment string
	Client      webhook.ClientService
	ClientAuth  runtime.ClientAuthInfoWriter
}

func (e *ReleaseManagerExporter) SendSuccessfulReleaseEvent(ctx context.Context, event models.DaemonKubernetesDeploymentWebhookRequest) error {
	event.Environment = e.Environment
	e.Log.With("event", event).Infof("SuccesfulRelease Event")

	_, err := e.Client.PostWebhookDaemonK8sDeploy(webhook.NewPostWebhookDaemonK8sDeployParams().WithBody(&event), e.ClientAuth)
	if err != nil {
		return err
	}
	return nil
}

func (e *ReleaseManagerExporter) SendPodErrorEvent(ctx context.Context, event models.DaemonKubernetesErrorWebhookRequest) error {
	event.Environment = e.Environment
	e.Log.With("event", event).Infof("PodError Event")

	_, err := e.Client.PostWebhookDaemonK8sError(webhook.NewPostWebhookDaemonK8sErrorParams().WithBody(&event), e.ClientAuth)
	if err != nil {
		return err
	}
	return nil
}

func (e *ReleaseManagerExporter) SendJobErrorEvent(ctx context.Context, event models.DaemonKubernetesJobErrorWebhookRequest) error {
	event.Environment = e.Environment
	e.Log.With("event", event).Infof("JobError Event")
	_, err := e.Client.PostWebhookDaemonK8sJoberror(webhook.NewPostWebhookDaemonK8sJoberrorParams().WithBody(&event), e.ClientAuth)
	if err != nil {
		return err
	}
	return nil
}
