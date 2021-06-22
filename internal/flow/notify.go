package flow

import (
	"context"

	"github.com/lunarway/release-manager/generated/http/models"
	"github.com/pkg/errors"
)

func (s *Service) NotifyK8SDeployEvent(ctx context.Context, event *models.DaemonKubernetesDeploymentWebhookRequest) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.NotifyK8SDeployment")
	defer span.Finish()
	span, _ = s.Tracer.FromCtx(ctx, "post k8s deploy slack message")
	err := s.Slack.NotifyK8SDeployEvent(ctx, event)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "post k8s deploy slack message")
	}
	return nil
}

func (s *Service) NotifyK8SPodErrorEvent(ctx context.Context, event *models.DaemonKubernetesErrorWebhookRequest) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.NotifyK8SPodErrorEvent")
	defer span.Finish()
	span, _ = s.Tracer.FromCtx(ctx, "post k8s NotifyK8SPodErrorEvent slack message")
	err := s.Slack.NotifyK8SPodErrorEvent(ctx, event)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "post k8s NotifyK8SPodErrorEvent slack message")
	}
	return nil
}

func (s *Service) NotifyK8SJobErrorEvent(ctx context.Context, event *models.DaemonKubernetesJobErrorWebhookRequest) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.NotifyK8SJobErrorEvent")
	defer span.Finish()
	span, _ = s.Tracer.FromCtx(ctx, "post k8s NotifyK8SJobErrorEvent slack message")
	err := s.Slack.NotifyK8SJobErrorEvent(ctx, event)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "post k8s NotifyK8SJobErrorEvent slack message")
	}
	return nil
}
