package broker

import (
	"context"
	"errors"
)

// Broker is capable of publishing and consuming Publishable messages.
type Broker interface {
	// Publish publishes a Publishable message on the broker.
	Publish(ctx context.Context, message Publishable) error
	// StartConsumer consumes messages on a broker. This method is blocking and
	// will always return with ErrBrokerClosed after calls to Close.
	StartConsumer(handlers map[string]func([]byte) error, errorHandler func(msgType string, msgBody []byte, err error)) error
	// Close closes the broker.
	Close() error
}

// Publishable represents an enty capable of being published and consumed by a
// Broker.
type Publishable interface {
	Type() string
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

// ErrBrokerClosed indicates that the broker was closed by a call to Close.
var ErrBrokerClosed = errors.New("broker: broker closed")
