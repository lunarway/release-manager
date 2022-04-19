package amqpextra

import (
	"context"

	internalamqp "github.com/lunarway/release-manager/internal/amqp"
	"github.com/lunarway/release-manager/internal/broker"
	amqp "github.com/rabbitmq/amqp091-go"
)

// StartConsumer consumes messages from an AMQP queue. The method is blocking
// and will always return with ErrBrokerClosed after calls to Close. In case an
// event is dropped after failed handlings and redelivery func eventDropped is
// called.
//
// The workers configured queue is declared on startup along with a binding to
// the exchange with routing key.
func (w *Worker) StartConsumer(handlers map[string]func([]byte) error, eventDropped func(msgType string, msgBody []byte, err error)) error {
	m := &mux{
		handlers:     handlers,
		log:          w.config.Logger,
		eventDropped: eventDropped,
	}

	consumer := internalamqp.ConsumerConfig{
		Exchange:        w.config.Exchange,
		Queue:           w.config.Queue,
		DurableQueue:    true,
		RoutingPatterns: []string{w.config.RoutingKey},
		Prefetch:        0,
		Handle: func(message *amqp.Delivery) error {
			return m.ServeMsg(context.Background(), *message)
		},
		WorkerCount: 1,
	}

	consumersStarted := make(chan struct{})
	err := w.worker.StartConsumer([]internalamqp.ConsumerConfig{consumer}, consumersStarted)
	if err != nil {
		return err
	}

	return broker.ErrBrokerClosed
}
