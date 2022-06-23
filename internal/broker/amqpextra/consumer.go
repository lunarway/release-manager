package amqpextra

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	internalamqp "github.com/lunarway/release-manager/internal/amqp"
	"github.com/lunarway/release-manager/internal/broker"
	amqp "github.com/rabbitmq/amqp091-go"
	"k8s.io/utils/strings"
)

// StartConsumer consumes messages from an AMQP queue. The method is blocking
// and will always return with ErrBrokerClosed after calls to Close. In case an
// event is dropped after failed handlings and redelivery func eventDropped is
// called.
//
// The workers configured queue is declared on startup along with a binding to
// the exchange with routing key.
func (w *Worker) StartConsumer(handlers map[string]func([]byte) error, fanoutHandlers map[string]func([]byte) error, eventDropped func(msgType string, msgBody []byte, err error)) error {
	m := &mux{
		handlers:     handlers,
		log:          w.config.Logger,
		eventDropped: eventDropped,
	}

	fanoutID := strings.ShortenString(uuid.New().String(), 5)
	consumers := []internalamqp.ConsumerConfig{
		{
			Exchange:            w.config.Exchange,
			DurableExchange:     true,
			AutoDeletedExchange: false,

			Queue:           w.config.Queue,
			ExclusiveQueue:  false,
			DurableQueue:    true,
			RoutingPatterns: []string{w.config.RoutingKey},
			Prefetch:        0,
			Handle: func(message *amqp.Delivery) error {
				return m.ServeMsg(context.Background(), *message)
			},
			WorkerCount: 1,
		},
		{
			Exchange:            fmt.Sprintf("%s-git-fanout", w.config.Exchange),
			DurableExchange:     false,
			AutoDeletedExchange: true,
			Queue:               fmt.Sprintf("%s-git-fanout-%s", w.config.Queue, fanoutID),
			ExclusiveQueue:      true,
			DurableQueue:        false,
			Prefetch:            0,
			RoutingPatterns:     []string{"#"},
			Handle: func(message *amqp.Delivery) error {
				return nil
			},
			WorkerCount: 1,
		},
	}

	consumersStarted := make(chan struct{})
	err := w.worker.StartConsumer(consumers, consumersStarted)
	if err != nil {
		return err
	}

	return broker.ErrBrokerClosed
}
