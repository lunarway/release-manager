package amqpextra

import (
	"context"
	"strings"

	"github.com/lunarway/release-manager/internal/amqp"
	"github.com/lunarway/release-manager/internal/broker"
	"github.com/lunarway/release-manager/internal/tracing"
)

// Publish publishes a Publishable message to AMQP. It will block until the
// message is confirmed on the server. In case of connection failures the method
// will block until connection is restored and message has been confirmed on the
// server.
//
// If ctx is cancelled before completing the publish the operation is cancelled.
func (w *Worker) Publish(ctx context.Context, message broker.Publishable) error {
	correlationID := tracing.RequestIDFromContext(ctx)

	err := w.worker.Publish(ctx, amqp.PublishDto{
		Exchange:      w.config.Exchange,
		RoutingKey:    strings.Join([]string{w.config.RoutingKey, message.Type()}, "."),
		MessageType:   message.Type(),
		CorrelationID: correlationID,
		Message:       message,
	})
	if err != nil {
		return err
	}
	return nil
}
