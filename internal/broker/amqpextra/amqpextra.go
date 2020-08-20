package amqpextra

import (
	"time"

	"github.com/makasim/amqpextra"
	"github.com/streadway/amqp"
)

// Worker is a RabbitMQ consumer and publisher. It configures an AMQP channel to
// consume messages from a queue along with a publisher to publish messages to
// an exchange.
//
// It transparently recovers from connection failures and publishes are blocked
// during recovery.
type Worker struct {
	conn      *amqpextra.Connection
	consumer  *amqpextra.Consumer
	publisher *amqpextra.Publisher
	config    Config
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

	conn := amqpextra.DialConfig([]string{c.Connection.Raw()}, amqp.Config{
		Heartbeat: 60 * time.Second,
		Vhost:     c.Connection.VirtualHost,
	})
	conn.SetLogger(newLogger(c.Logger))
	conn.SetReconnectSleep(c.ReconnectionTimeout)

	<-conn.Ready()

	w := &Worker{
		conn:      conn,
		config:    c,
		publisher: conn.Publisher(),
	}
	err := w.declareExchange()
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Worker) declareExchange() error {
	amqpConn, err := w.conn.Conn()
	if err != nil {
		return err
	}
	channel, err := amqpConn.Channel()
	if err != nil {
		return err
	}
	err = channel.ExchangeDeclare(
		w.config.Exchange,
		"topic", // kind
		true,    // durable
		false,   // autoDelete
		false,   // internal
		false,   // noWait
		nil,     // args
	)
	if err != nil {
		return err
	}
	return nil
}

// Close closes the worker stopping any active consumers and publishes.
func (w *Worker) Close() error {
	w.conn.Close()
	return nil
}
