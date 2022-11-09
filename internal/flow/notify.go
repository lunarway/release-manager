package flow

import (
	"context"
	"fmt"

	"github.com/lunarway/release-manager/internal/http"

	"github.com/pkg/errors"
)

func (s *Service) NotifyK8SDeployEvent(ctx context.Context, event *http.ReleaseEvent) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.NotifyK8SDeployment")
	defer span.Finish()
	if s.NotifyReleaseSucceededHook != nil {
		go s.NotifyReleaseSucceededHook(noCancel{ctx: ctx}, NotifyReleaseSucceededOptions{
			Name:          event.Name,
			Namespace:     event.Namespace,
			ResourceType:  event.ResourceType,
			AvailablePods: event.AvailablePods,
			DesiredPods:   event.DesiredPods,
			ArtifactID:    event.ArtifactID,
			AuthorEmail:   event.AuthorEmail,
			Environment:   event.Environment,
		})
	}

	return nil
}

func (s *Service) NotifyK8SPodErrorEvent(ctx context.Context, event *http.PodErrorEvent) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.NotifyK8SPodErrorEvent")
	defer span.Finish()
	span, _ = s.Tracer.FromCtx(ctx, "post k8s NotifyK8SPodErrorEvent slack message")
	err := s.Slack.NotifyK8SPodErrorEvent(ctx, event)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "post k8s NotifyK8SPodErrorEvent slack message")
	}
	if s.NotifyReleaseFailedHook != nil {

		go s.NotifyReleaseFailedHook(noCancel{ctx: ctx}, NotifyReleaseFailedOptions{
			PodName:     event.PodName,
			Namespace:   event.Namespace,
			Errors:      event.ErrorStrings(),
			AuthorEmail: event.AuthorEmail,
			Environment: event.Environment,
			ArtifactID:  event.ArtifactID,
			Squad:       event.Squad,
			AlertSquad:  event.AlertSquad,
		})
	}
	return nil
}

func (s *Service) NotifyK8SJobErrorEvent(ctx context.Context, event *http.JobErrorEvent) error {
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
