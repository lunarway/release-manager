package webhook

import (
	"context"

	"github.com/lunarway/release-manager/internal/broker"
	"gopkg.in/go-playground/webhooks.v5/github"
)

func Publisher(b broker.Broker) func(ctx context.Context, payload github.PushPayload) error {
	return func(ctx context.Context, payload github.PushPayload) error {
		return b.Publish(ctx, NewPayload(payload))
	}
}
