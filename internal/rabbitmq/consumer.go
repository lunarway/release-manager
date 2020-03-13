package rabbitmq

import (
	"fmt"

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

func (c *consumer) Start(logger *log.Logger, handler func([]byte) error) error {
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
		logger := logger.WithFields(
			"exchange", msg.Exchange,
			"routingKey", msg.RoutingKey,
			"messageId", msg.MessageId,
			"timestamp", msg.Timestamp,
			"headers", fmt.Sprintf("%#v", msg.Headers),
		)
		logger.Infof("Received message from exchange=%s routingKey=%s messageId=%s timestamp=%s", msg.Exchange, msg.RoutingKey, msg.MessageId, msg.Timestamp)
		err := handler(msg.Body)
		if err != nil {
			logger.WithFields("error", fmt.Sprintf("%+v", err)).Errorf("Failed to handle message: nacking and requeing: %v", err)
			err := msg.Nack(false, true)
			if err != nil {
				logger.WithFields("error", fmt.Sprintf("%+v", err)).Errorf("Failed to nack message: %v", err)
			}
			continue
		}
		err = msg.Ack(false)
		if err != nil {
			logger.Errorf("Failed to ack message: %v", err)
		}
	}
	return nil
}
