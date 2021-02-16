package amqpextra

import (
	"github.com/lunarway/release-manager/internal/amqp"
)

// Worker is a RabbitMQ consumer and publisher. It configures an AMQP channel to
// consume messages from a queue along with a publisher to publish messages to
// an exchange.
//
// It transparently recovers from connection failures and publishes are blocked
// during recovery.
type Worker struct {
	worker *amqp.Worker
	config Config
}

// New allocates, intializes and returns a new Worker instance.
func New(c Config) (*Worker, error) {
	logger := c.Logger.WithFields(
		"host", c.Connection.Host,
		"user", c.Connection.User,
		"port", c.Connection.Port,
		"virtualHost", c.Connection.VirtualHost,
		"exchange", c.Exchange,
		"queue", c.Queue,
		"prefetch", c.Prefetch,
		"reconnectionTimeout", c.ReconnectionTimeout,
	)

	logger.Infof("Connecting to: %s", c.Connection.String())
	worker, err := amqp.New(amqp.Config{
		Logger:                          c.Logger,
		ConnectionString:                c.Connection.Raw(),
		VirtualHost:                     c.Connection.VirtualHost,
		Prefix:                          "",
		ReconnectionTimeout:             c.ReconnectionTimeout,
		ConnectTimeout:                  c.ConnectionTimeout,
		InitTimeout:                     c.InitTimeout,
		MaxUnconfirmedInFlightPublishes: 1, // this enables publisher confirms
		OnDial: func(attempt int, err error) {
			c.Logger.Infof("Dialing to amqp attempt %d due error: %v", attempt, err)
		},
	})
	if err != nil {
		return nil, err
	}

	return &Worker{
		worker: worker,
		config: c,
	}, nil
}

// Close closes the worker stopping any active consumers and publishes.
func (w *Worker) Close() error {
	return w.worker.Close()
}
