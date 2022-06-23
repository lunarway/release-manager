package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/lunarway/release-manager/internal/broker"
	"github.com/lunarway/release-manager/internal/log"
)

type Broker struct {
	logger *log.Logger
	queue  chan broker.Publishable
}

// New allocates and returns an in-memory Broker with provided queue size.
func New(logger *log.Logger, queueSize int) *Broker {
	return &Broker{
		logger: logger,
		queue:  make(chan broker.Publishable, queueSize),
	}
}

func (b *Broker) Close() error {
	close(b.queue)
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

func (b *Broker) StartConsumer(handlers map[string]func([]byte) error, fanoutHandlers map[string]func([]byte) error, errorHandler func(msgType string, msgBody []byte, err error)) error {
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
