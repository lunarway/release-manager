package rabbitmq

import (
	"fmt"
	"time"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type consumer struct {
	channel *amqp.Channel
	queue   *amqp.Queue
}

func newConsumer(amqpConn *amqp.Connection, exchangeName, queueName, routingKey string, prefetch int) (*consumer, error) {
	channel, err := amqpConn.Channel()
	if err != nil {
		return nil, errors.WithMessage(err, "get channel")
	}

	if prefetch > 0 {
		err := channel.Qos(prefetch, 0, false)
		if err != nil {
			return nil, errors.WithMessage(err, "set prefetch with qos")
		}
	}

	err = declareExchange(channel, exchangeName)
	if err != nil {
		return nil, errors.WithMessage(err, "declare exchange")
	}

	amqpQueue, err := channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return nil, errors.WithMessage(err, "declare queue")
	}

	err = channel.QueueBind(queueName, routingKey, exchangeName, false, nil)
	if err != nil {
		return nil, errors.WithMessage(err, "bind to queue")
	}
	return &consumer{
		channel: channel,
		queue:   &amqpQueue,
	}, nil
}

func (c *consumer) Close() error {
	// TODO: stop consuming messages with c.channel.Cancel() and wait for the
	// consumer to complete before closing the connection to avoid dropping
	// inflight messages.
	err := c.channel.Close()
	if err != nil {
		return errors.WithMessage(err, "close amqp channel")
	}
	return nil
}

func (c *consumer) Start(logger *log.Logger, handlers map[string]func([]byte) error) error {
	msgs, err := c.channel.Consume(
		c.queue.Name,      // queue
		"release-manager", // consumer
		false,             // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)
	if err != nil {
		return errors.WithMessage(err, "consume messages")
	}

	for msg := range msgs {
		logger := logger.With(
			"routingKey", msg.RoutingKey,
			"messageId", msg.MessageId,
			"eventType", msg.Type,
			"correlationId", msg.CorrelationId,
			"headers", fmt.Sprintf("%#v", msg.Headers),
		)
		logger.Infof("Received message from exchange=%s routingKey=%s messageId=%s timestamp=%s", msg.Exchange, msg.RoutingKey, msg.MessageId, msg.Timestamp)
		now := time.Now()

		handler, ok := handlers[msg.Type]
		if !ok {
			logger.With("res", map[string]interface{}{
				"status": "failed",
				"error":  "unprocessable",
			}).Errorf("[consumer] [UNPROCESSABLE] Failed to handle message: no handler registered for event type '%s': dropping it", msg.Type)
			err := msg.Nack(false, false)
			if err != nil {
				logger.Errorf("Failed to nack message: %v", err)
			}
			continue
		}
		err := handler(msg.Body)
		duration := time.Since(now).Milliseconds()
		if err != nil {
			logger.With("res", map[string]interface{}{
				"status":       "failed",
				"responseTime": duration,
				"error":        fmt.Sprintf("%+v", err),
			}).Errorf("[consumer] [FAILED] Failed to handle message: nacking and requeing: %v", err)
			// TODO: remove comments to allow for redelivery. This will put events
			// into the unacknowledged state

			// err := msg.Nack(false, true) if err != nil {
			//  logger.WithFields("error", fmt.Sprintf("%+v", err)).Errorf("Failed to nack message: %v", err)
			// }
			continue
		}
		logger.With("res", map[string]interface{}{
			"status":       "ok",
			"responseTime": duration,
		}).Info("[OK] Event handled successfully")
		err = msg.Ack(false)
		if err != nil {
			logger.Errorf("Failed to ack message: %v", err)
		}
	}
	return nil
}
