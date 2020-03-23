package flow

import (
	"context"

	"strings"

	"github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
)

func (s *Service) NotifyCommitter(ctx context.Context, event *http.PodNotifyRequest) error {
	span, ctx := s.Tracer.FromCtx(ctx, "flow.NotifyCommitter")
	defer span.Finish()
	email := event.AuthorEmail
	if !strings.Contains(email, "@lunar.app") {
		//check UserMappings
		lwEmail, ok := s.UserMappings[email]
		if !ok {
			log.WithContext(ctx).Errorf("%s is not a Lunar Way email and no mapping exist", email)
			return errors.Errorf("%s is not a Lunar Way email and no mapping exist", email)
		}
		email = lwEmail
	}
	span, _ = s.Tracer.FromCtx(ctx, "post private slack message")
	err := s.Slack.PostPrivateMessage(ctx, email, event)
	span.Finish()
	if err != nil {
		return errors.WithMessage(err, "post private message")
	}

	return nil
}
