package amqpextra

import (
	"github.com/lunarway/release-manager/internal/broker"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

// StartConsumer consumes messages from an AMQP queue. The method is blocking
// and will always return with ErrBrokerClosed after calls to Close. In case an
// event is dropped after failed handlings and redelivery func eventDropped is
// called.
//
// The workers configured queue is declared on startup along with a binding to
// the exchange with routing key.
func (w *Worker) StartConsumer(handlers map[string]func([]byte) error, eventDropped func(msgType string, msgBody []byte, err error)) error {
	w.consumer = w.conn.Consumer(w.config.Queue, &mux{
		handlers:     handlers,
		log:          w.config.Logger,
		eventDropped: eventDropped,
	})
	w.consumer.Use(loggerMiddleware(w.config.Logger))
	defer w.consumer.Close()

	w.consumer.SetInitFunc(func(conn *amqp.Connection) (*amqp.Channel, <-chan amqp.Delivery, error) {
		channel, err := conn.Channel()
		if err != nil {
			return nil, nil, errors.WithMessage(err, "create channel")
		}
		_, err = channel.QueueDeclare(w.config.Queue, true, false, false, false, nil)
		if err != nil {
			return nil, nil, errors.WithMessage(err, "declare queue")
		}

		err = channel.QueueBind(w.config.Queue, w.config.RoutingKey, w.config.Exchange, false, nil)
		if err != nil {
			return nil, nil, errors.WithMessage(err, "bind to queue")
		}

		err = channel.Qos(w.config.Prefetch, 0, false)
		if err != nil {
			return nil, nil, errors.WithMessage(err, "set qos on queue")
		}

		msgCh, err := channel.Consume(
			w.config.Queue,
			"",
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return nil, nil, errors.WithMessage(err, "consume messages")
		}

		return channel, msgCh, nil
	})
	w.consumer.Run()

	return broker.ErrBrokerClosed
}
