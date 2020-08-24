package amqpextra

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lunarway/release-manager/internal/broker"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/makasim/amqpextra"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

// Publish publishes a Publishable message to AMQP. It will block until the
// message is confirmed on the server. In case of connection failures the method
// will block until connection is restored and message has been confirmed on the
// server.
//
// If ctx is cancelled before completing the publish the operation is cancelled.
func (w *Worker) Publish(ctx context.Context, message broker.Publishable) error {
	correlationID := tracing.RequestIDFromContext(ctx)
	messageType := message.Type()
	messageID, err := uuid.NewRandom()
	if err != nil {
		w.config.Logger.Errorf("Failed to create a correlation ID. Continues execution: %v", err)
	}
	body, err := message.Marshal()
	if err != nil {
		return errors.WithMessage(err, "marshal message")
	}

	logger := w.config.Logger.With(
		"body", message,
		"routingKey", w.config.RoutingKey,
		"messageId", messageID,
		"eventType", messageType,
		"correlationId", correlationID,
	)
	logger.Infof("[publisher] Publishing message type=%s from exchange=%s routingKey=%s messageId=%s correlationId=%s", messageType, w.config.Exchange, w.config.RoutingKey, messageID, correlationID)

	now := time.Now()

	resultCh := make(chan error, 1)
	w.publisher.Publish(amqpextra.Publishing{
		Context:   ctx,
		Exchange:  w.config.Exchange,
		Key:       w.config.RoutingKey,
		Immediate: false,
		Mandatory: false,
		WaitReady: true, // wait for the publisher to be ready instead of returning an error.
		ResultCh:  resultCh,
		Message: amqp.Publishing{
			Type:          message.Type(),
			Body:          body,
			MessageId:     messageID.String(),
			CorrelationId: correlationID,
			ContentType:   "application/json",
			DeliveryMode:  amqp.Persistent, // this ensures messages are persisted to disk
		},
	})
	err = <-resultCh

	duration := time.Since(now).Milliseconds()

	if err != nil {
		logger.With(
			"messageId", messageID,
			"eventType", messageType,
			"correlationId", tracing.RequestIDFromContext(ctx),
			"res", map[string]interface{}{
				"status":       "failed",
				"responseTime": duration,
				"error":        err,
			}).Errorf("[publisher] [FAILED] Failed to publish message: %v", err)
		return errors.WithMessage(err, "publish message")
	}

	logger.With(
		"messageId", messageID,
		"eventType", messageType,
		"correlationId", tracing.RequestIDFromContext(ctx),
		"res", map[string]interface{}{
			"status":       "ok",
			"responseTime": duration,
		}).Info("[publisher] [OK] Published message successfully")
	return nil
}
