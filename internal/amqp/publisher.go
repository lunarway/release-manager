package amqp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type rawPublisher struct {
	channel      *amqp.Channel
	exchangeName string
	routingKey   string
}

func newPublisher(amqpConn *amqp.Connection, exchangeName string) (publisher, error) {
	channel, err := amqpConn.Channel()
	if err != nil {
		return nil, errors.WithMessage(err, "get channel")
	}
	return &rawPublisher{
		channel:      channel,
		exchangeName: exchangeName,
	}, nil
}

func (p *rawPublisher) Publish(ctx context.Context, eventType, messageID string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return errors.WithMessage(err, "marshal message")
	}
	pub := amqp.Publishing{
		ContentType:   "application/json",
		Type:          eventType,
		Body:          data,
		MessageId:     messageID,
		CorrelationId: tracing.RequestIDFromContext(ctx),
	}
	// TODO: add publisher confirms to the channel and wait for acknowledgement
	// before returning
	err = p.channel.Publish(p.exchangeName, p.routingKey, false, false, pub)
	if err != nil {
		return err
	}
	return nil
}

func (p *rawPublisher) Close() error {
	err := p.channel.Close()
	if err != nil {
		return errors.WithMessage(err, "close amqp channel")
	}
	return nil
}

type loggingPublisher struct {
	publisher publisher
	logger    *log.Logger
}

func (p *loggingPublisher) Publish(ctx context.Context, eventType, messageID string, message interface{}) error {
	logger := p.logger.WithContext(ctx).WithFields("body", message)
	logger.Debug("Publishing message")
	now := time.Now()
	err := p.publisher.Publish(ctx, eventType, messageID, message)
	duration := time.Since(now).Milliseconds()
	if err != nil {
		logger.With(
			"messageId", messageID,
			"eventType", eventType,
			"correlationId", tracing.RequestIDFromContext(ctx),
			"res", map[string]interface{}{
				"status":       "failed",
				"responseTime": duration,
				"error":        err,
			}).Errorf("[publisher] [FAILED] Failed to publish message: %v", err)
		return err
	}
	logger.With(
		"messageId", messageID,
		"eventType", eventType,
		"correlationId", tracing.RequestIDFromContext(ctx),
		"res", map[string]interface{}{
			"status":       "ok",
			"responseTime": duration,
		}).Info("[publisher] [OK] Published message successfully")
	return nil
}

func (p *loggingPublisher) Close() error {
	return p.publisher.Close()
}
