package amqp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/makasim/amqpextra/publisher"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Publish publishes a message over AMQP. It will block until the message
// is confirmed on the server. In case of connection failures the method will
// block until connection is restored and message has been confirmed on the
// server.
//
// If ctx is cancelled before completing the publish the operation is cancelled.
func (w *Worker) Publish(ctx context.Context, dto PublishDto) error {
	prefixedExchange := w.prefixed(dto.Exchange)
	exchangeType := dto.ExchangeType
	if exchangeType == "" {
		exchangeType = amqp.ExchangeTopic
	}
	err := w.ensureExchangeDeclared(ctx, prefixedExchange, exchangeType)
	if err != nil {
		return errors.WithMessage(err, "declare exchange")
	}

	body, err := json.Marshal(dto.Message)
	if err != nil {
		return errors.WithMessage(err, "marshal message")
	}

	timestamp := time.Now()
	messageID, err := uuid.NewRandom()
	if err != nil {
		w.logger.Errorf("[amqp] Failed to create a correlation ID. Continues execution: %v", err)
	}

	loggingContext := fmt.Sprintf(
		"type='%s' exchange='%s' routingKey='%s' messageId='%s' correlationId='%s' timestamp='%s' publishMode='%s'",
		dto.MessageType, prefixedExchange, dto.RoutingKey, messageID, dto.CorrelationID, timestamp, w.PublishMode)
	w.logger.Infof("[amqp] Publishing message %s", loggingContext)
	w.logger.Debugf("[amqp] Publishing message payload: %+v", dto.Message)

	now := time.Now()

	// when ctx is cancelled this unblocks, so we do not have to handle timeouts
	// here but instead we rely on the caller to have set a timeout.
	err = w.publisher.Publish(publisher.Message{
		Context:      ctx,
		Exchange:     prefixedExchange,
		Key:          dto.RoutingKey,
		Immediate:    false,
		Mandatory:    false,
		ErrOnUnready: false,
		Publishing: amqp.Publishing{
			Type:          dto.MessageType,
			Body:          body,
			MessageId:     messageID.String(),
			CorrelationId: dto.CorrelationID,
			ContentType:   "application/json",
			Timestamp:     timestamp,
			// TODO: make this configurable
			// DeliveryMode:  amqp.Persistent, // this ensures messages are persisted to disk
		},
	})

	duration := time.Since(now).Milliseconds()

	if err != nil {
		w.logger.Errorf("[amqp] [FAILED] Failed to publish message %s status='failed' responseTime='%d' error='%v'", loggingContext, duration, err)
		return errors.WithMessage(err, "publish message")
	}

	w.logger.Infof("[amqp] [OK] Published message successfully %s status='ok' responseTime='%d'", loggingContext, duration)
	return nil
}

// ensureExchangeDeclared ensures that an exchange named prefixedExchange is
// declared with the given kind. Once it is declared for an exchange it becomes
// a noop.
func (w *Worker) ensureExchangeDeclared(ctx context.Context, prefixedExchange, kind string) error {
	_, ok := w.declaredExchanges[prefixedExchange]
	if ok {
		w.logger.Debugf("[amqp] Exchange '%s' already declared", prefixedExchange)
		return nil
	}
	w.logger.Debugf("[amqp] Declaring publishing exchange '%s' of type '%s'", prefixedExchange, kind)
	err := w.declareExchange(prefixedExchange, kind)
	if err != nil {
		return err
	}
	w.declaredExchanges[prefixedExchange] = struct{}{}
	return nil
}

func (w *Worker) declareExchange(exchange, kind string) error {
	amqpConn, err := w.dialer.Connection(context.Background())
	if err != nil {
		return err
	}

	channel, err := amqpConn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	err = channel.ExchangeDeclare(
		exchange,
		kind,  // kind
		true,  // durable
		false, // autoDelete
		false, // internal
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return err
	}
	return nil
}

type PublishDto struct {
	Exchange      string
	RoutingKey    string
	MessageType   string
	CorrelationID string
	Message       interface{}
	// ExchangeType is the kind of exchange to declare for this publish (e.g.
	// "topic" or "fanout"). When empty it defaults to "topic".
	ExchangeType string
}
