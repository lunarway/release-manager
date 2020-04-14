package amqp

import (
	"context"
	"time"

	"github.com/lunarway/release-manager/internal/broker"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/tracing"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type rawPublisher struct {
	channel          *amqp.Channel
	exchangeName     string
	routingKey       string
	republishTimeout time.Duration

	// close is used to signal that the publisher should close and any resends
	// should be stopped.
	close chan struct{}
	// closed is set to true
	closed        bool
	confirmations chan amqp.Confirmation
	republishing  func(ctx context.Context, reason error)
}

func newPublisher(amqpConn *amqp.Connection, exchangeName string, republishTimeout time.Duration, republishing func(context.Context, error)) (*rawPublisher, error) {
	channel, err := amqpConn.Channel()
	if err != nil {
		return nil, errors.WithMessage(err, "get channel")
	}
	err = channel.Confirm(false)
	if err != nil {
		return nil, errors.WithMessage(err, "enable confirm mode")
	}
	confirmations := make(chan amqp.Confirmation, 1)
	channel.NotifyPublish(confirmations)
	return &rawPublisher{
		channel:          channel,
		exchangeName:     exchangeName,
		close:            make(chan struct{}),
		confirmations:    confirmations,
		republishing:     republishing,
		republishTimeout: republishTimeout,
	}, nil
}

func (p *rawPublisher) Publish(ctx context.Context, eventType, messageID string, message []byte) error {
	pub := amqp.Publishing{
		ContentType:   "application/json",
		Type:          eventType,
		Body:          message,
		MessageId:     messageID,
		CorrelationId: tracing.RequestIDFromContext(ctx),
	}
	for {
		err := p.channel.Publish(p.exchangeName, p.routingKey, false, false, pub)
		if err != nil {
			// TODO: handle lost channels here. If the channel gets closed, for some
			// reason, we will keep retrying

			// retry the publish after a timeout or stop on context cancellation or if
			// the publisher is closed
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-p.close:
				return broker.ErrBrokerClosed
			case <-time.After(p.republishTimeout):
				p.republishing(ctx, err)
				continue
			}
		}
		// wait for confirmation of the publish but respect context cancellation and
		// if the publisher is closed
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.close:
			return broker.ErrBrokerClosed
		case confirm, ok := <-p.confirmations:
			// the confirmations channel can be closed if the amqp channel is closed
			// unexpectedly and this the read will be possible. We do not expect this
			// case nor handle it in the publisher so this is just a fail guard until
			// prober handling is in place. Note that this will only be triggered if
			// the AMQP channel is closed, not the connection. If the connection is
			// closed the whole publisher will be replaced.
			if !ok {
				return errors.New("confirmation channel closed")
			}
			if confirm.Ack {
				return nil
			}
		case <-time.After(p.republishTimeout):
			p.republishing(ctx, errors.New("published timed out"))
		}
	}
}

func (p *rawPublisher) Close() error {
	if p.closed {
		return nil
	}
	close(p.close)
	p.closed = true
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

func (p *loggingPublisher) Publish(ctx context.Context, eventType, messageID string, message []byte) error {
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
