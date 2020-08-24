package amqpextra

import (
	"context"
	"fmt"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/makasim/amqpextra"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

// mux implements interface amqpextra.Worker  and facilitates message
// multiplexing to message handlers based on message type.
type mux struct {
	handlers     map[string]func([]byte) error
	log          *log.Logger
	eventDropped func(msgType string, msgBody []byte, err error)
}

var _ amqpextra.Worker = &mux{}

func (m mux) ServeMsg(ctx context.Context, msg amqp.Delivery) interface{} {
	handler, ok := m.handlers[msg.Type]
	if !ok {
		m.nack(msg, false, "no handler")
		return fmt.Errorf("unprocessable event type '%s': event dropped", msg.Type)
	}

	err := handler(msg.Body)
	if err != nil {
		if msg.Redelivered {
			m.eventDropped(msg.Type, msg.Body, err)
			m.nack(msg, false, "nack without requeue due redelivery failed")
			return errors.WithMessage(err, "messaged dropped")
		}
		m.nack(msg, true, "nack with requeue failed")
		return errors.WithMessage(err, "message requeued")
	}
	m.ack(msg)
	return nil
}

func (m mux) ack(msg amqp.Delivery) {
	err := msg.Nack(false, false)
	if err != nil {
		m.log.Errorf("ack failed for event '%s': %v", msg.Type, err)
	}
}

func (m mux) nack(msg amqp.Delivery, requeue bool, reason string) {
	err := msg.Nack(false, requeue)
	if err != nil {
		m.log.Errorf("%s for event '%s': %v", reason, msg.Type, err)
	}
}
