package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/lunarway/release-manager/internal/broker"
	"github.com/lunarway/release-manager/internal/log"
)

type Broker struct {
	logger         *log.Logger
	queue          chan broker.Publishable
	broadcastQueue chan broker.Publishable
}

// New allocates and returns an in-memory Broker with provided queue size.
func New(logger *log.Logger, queueSize int) *Broker {
	return &Broker{
		logger:         logger,
		queue:          make(chan broker.Publishable, queueSize),
		broadcastQueue: make(chan broker.Publishable, queueSize),
	}
}

func (b *Broker) Close() error {
	close(b.queue)
	close(b.broadcastQueue)
	return nil
}

func (b *Broker) Publish(ctx context.Context, event broker.Publishable) error {
	b.logger.WithFields("message", event).Info("Publishing message")
	now := time.Now()
	b.queue <- event
	duration := time.Since(now).Milliseconds()
	b.logger.With(
		"eventType", event.Type(),
		"res", map[string]interface{}{
			"status":       "ok",
			"responseTime": duration,
		}).Info("[publisher] [OK] Published message successfully")
	return nil
}

// PublishBroadcast publishes a message to the in-process broadcast queue. In a
// single-process broker the only replica is this process, so the message is
// delivered to this process' broadcast consumer.
func (b *Broker) PublishBroadcast(ctx context.Context, event broker.Publishable) error {
	b.logger.WithFields("message", event).Info("Publishing broadcast message")
	b.broadcastQueue <- event
	return nil
}

// StartBroadcastConsumer consumes messages from the in-process broadcast queue.
// It is blocking and returns ErrBrokerClosed once the broker is closed.
func (b *Broker) StartBroadcastConsumer(handler func([]byte) error) error {
	for msg := range b.broadcastQueue {
		logger := b.logger.With("eventType", msg.Type())
		body, err := msg.Marshal()
		if err != nil {
			logger.Errorf("[broadcast] Could not get body of message: %v", err)
			continue
		}
		if err := handler(body); err != nil {
			logger.Errorf("[broadcast] Failed to handle message: %v", err)
		}
	}
	return broker.ErrBrokerClosed
}

func (b *Broker) StartConsumer(handlers map[string]func([]byte) error, errorHandler func(msgType string, msgBody []byte, err error)) error {
	for msg := range b.queue {
		logger := b.logger.With(
			"eventType", msg.Type(),
		)
		logger.WithFields("message", msg).Infof("Received message type=%s", msg.Type())
		handler, ok := handlers[msg.Type()]
		if !ok {
			logger.With("res", map[string]interface{}{
				"status": "failed",
				"error":  "unprocessable",
			}).Errorf("[consumer] [UNPROCESSABLE] Failed to handle message: no handler registered for event type '%s': dropping it", msg.Type)
			continue
		}
		body, err := msg.Marshal()
		if err != nil {
			logger.Errorf("[consumer] [UNPROCESSABLE] Could not get body of message: %v", err)
			continue
		}
		now := time.Now()
		err = handler(body)
		duration := time.Since(now).Milliseconds()
		if err != nil {
			logger.With("res", map[string]interface{}{
				"status":       "failed",
				"responseTime": duration,
				"error":        fmt.Sprintf("%+v", err),
			}).Errorf("[consumer] [FAILED] Failed to handle message: trigger error handling and acking: %v", err)
			errorHandler(msg.Type(), body, err)
			continue
		}
		logger.With("res", map[string]interface{}{
			"status":       "ok",
			"responseTime": duration,
		}).Info("[OK] Event handled successfully")
	}
	return broker.ErrBrokerClosed
}
