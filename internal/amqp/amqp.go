package amqp

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

// Worker is a RabbtiMQ consumer and publisher. It will setup an AMQP channel
// to consume messages from an exchange through a queue and will withstand
// disconnects on connection to RabbtiMQ.
//
// Reconnection is implemented as a chan *connection that is consumed in the
// Start method. If connection loss is detected the reconnector Go routine will
// setup a new connection and push it on to the channel thus keeping Start
// blocking.
type Worker struct {
	config Config

	// currentConsumer provides an active connection to consume from.
	currentConsumer chan *consumer
	// currentConsumer provides an active connection to publish on.
	currentPublisher publisher
	// shutdown is used to terminate the different Go routines in the worker. It
	// will be closed as a signal to stop.
	shutdown chan struct{}
	// connectionClosed is used to signal that a connection was lost and that the
	// reconnector should attempt to reestablish it.
	connectionClosed chan *amqp.Error
	// fatalError is used to signal that the worker was unable to recreate a
	// connection and that the worker should stop consuming and publishing
	// messages.
	fatalError chan error
}

type publisher interface {
	Publish(ctx context.Context, eventType, messageID string, message interface{}) error
	Close() error
}

type Config struct {
	Connection              ConnectionConfig
	Exchange                string
	Queue                   string
	RoutingKey              string
	Prefetch                int
	MaxReconnectionAttempts int
	ReconnectionTimeout     time.Duration
	Handlers                map[string]func([]byte) error
	AMQPConfig              *amqp.Config
	Logger                  *log.Logger
}

type ConnectionConfig struct {
	Host        string
	User        string
	Password    string
	VirtualHost string
	Port        int
}

func (c *ConnectionConfig) Raw() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/%s", c.User, c.Password, c.Host, c.Port, c.VirtualHost)
}

func (c *ConnectionConfig) String() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/%s", c.User, "***", c.Host, c.Port, c.VirtualHost)
}

// NewWorker allocates and returns a Worker consuming and publising messages on
// a RabbitMQ exchange.
func NewWorker(c Config) (*Worker, error) {
	worker := Worker{
		shutdown:         make(chan struct{}),
		currentConsumer:  make(chan *consumer),
		fatalError:       make(chan error),
		connectionClosed: make(chan *amqp.Error),
		config:           c,
	}
	worker.config.Logger = c.Logger.WithFields(
		"host", c.Connection.Host,
		"user", c.Connection.User,
		"port", c.Connection.Port,
		"virtualHost", c.Connection.VirtualHost,
		"exchange", c.Exchange,
		"queue", c.Queue,
		"prefetch", c.Prefetch,
		"maxReconnectionAttempts", c.MaxReconnectionAttempts,
		"reconnectionTimeout", c.ReconnectionTimeout,
		"amqpConfig", fmt.Sprintf("%#v", c.AMQPConfig),
	)

	err := worker.connect()
	if err != nil {
		return nil, err
	}
	go worker.reconnector()
	return &worker, nil
}

func (s *Worker) Close() error {
	close(s.shutdown)
	return nil
}

// ErrWorkerClosed is returned by the Consumer's Start method after a call to
// Close.
var ErrWorkerClosed = errors.New("rabbitmq: Worker closed")

// StartConsumer starts the consumer on the worker. The method is blocking and
// will only return if the worker is stopped with Close or the connection is
// lost and cannot be recreated.
func (s *Worker) StartConsumer() error {
	for {
		select {
		// worker is instructed to shutdown from a Close call
		case <-s.shutdown:
			return ErrWorkerClosed

		// worker is unable to recover a network failure and should stop
		case err := <-s.fatalError:
			return err

		// worker has a new connection that can be used to consume messages with the handler
		case conn := <-s.currentConsumer:
			// this call is blocking as long as the connection is available.
			err := conn.Start(s.config.Logger, s.config.Handlers)
			if err != nil {
				return err
			}
		}
	}
}

type Publishable interface {
	Type() string
	Body() interface{}
}

// Publish publishes a message on a configured RabbitMQ exchange.
func (s *Worker) Publish(ctx context.Context, event Publishable) error {
	uuid, err := uuid.NewRandom()
	if err != nil {
		s.config.Logger.Errorf("Failed to create a random message ID. Continue execution: %v", err)
	}
	err = s.currentPublisher.Publish(ctx, event.Type(), uuid.String(), event.Body())
	if err != nil {
		return err
	}
	return nil
}

func (s *Worker) connect() error {
	c := s.config
	c.Logger.Infof("Connecting to: %s", c.Connection.String())

	if c.AMQPConfig != nil && c.Connection.VirtualHost != c.AMQPConfig.Vhost {
		c.Logger.Infof("AMQP config overwrites provided virtual host")
	}
	if c.AMQPConfig == nil {
		c.AMQPConfig = &amqp.Config{
			Heartbeat: 60 * time.Second,
			Vhost:     c.Connection.VirtualHost,
		}
	}
	amqpConn, err := amqp.DialConfig(c.Connection.Raw(), *c.AMQPConfig)
	if err != nil {
		return errors.WithMessage(err, "connect to amqp")
	}
	connectionClosedListener := make(chan *amqp.Error)
	amqpConn.NotifyClose(connectionClosedListener)

	// TODO: this could be instantiated from a list of exchange and queue pairs to
	// setup more than one consumer
	consumer, err := newConsumer(amqpConn, c.Exchange, c.Queue, c.RoutingKey, c.Prefetch)
	if err != nil {
		return errors.WithMessage(err, "create consumer")
	}

	rawPublisher, err := newPublisher(amqpConn, c.Exchange)
	if err != nil {
		return errors.WithMessage(err, "create publisher")
	}
	s.currentPublisher = &loggingPublisher{
		publisher: rawPublisher,
		logger:    c.Logger,
	}

	// listen for connection failures on the specific connection along with
	// closing the connection if general shutdown is signalled
	go func() {
		c.Logger.Info("Connection close listener started")
		defer c.Logger.Info("Connection close listener stopped")
		select {
		case <-s.shutdown:
			if amqpConn.IsClosed() {
				return
			}
			err := consumer.Close()
			if err != nil {
				c.Logger.Errorf("Failed to close consumer: %v", err)
			}
			err = s.currentPublisher.Close()
			if err != nil {
				c.Logger.Errorf("Failed to close publisher: %v", err)
			}
			err = amqpConn.Close()
			if err != nil {
				c.Logger.Errorf("Failed to close amqp connection: %v", err)
			}
		case err, abnormalShutdown := <-connectionClosedListener:
			if !abnormalShutdown {
				c.Logger.Info("Connection closed due to normal shutdown")
				return
			}
			c.Logger.Info("Connection closed due to abnormal shutdown")
			// signal the worker that the connection was lost
			s.connectionClosed <- err
		}
	}()
	go func() {
		select {
		case <-s.shutdown:
		case s.currentConsumer <- consumer:
		}
	}()
	c.Logger.Info("Connected to RabbitMQ successfully")
	return nil
}

func (s *Worker) reconnector() {
	s.config.Logger.Info("Reconnector started")
	defer s.config.Logger.Info("Reconnector stopped")
	for {
		select {
		case <-s.shutdown:
			s.config.Logger.Info("Reconnector received shutdown signal")
			return
		case reason := <-s.connectionClosed:
			s.config.Logger.Info("Reconnector received connection closed signal")
			success := s.reconnect(reason)
			if !success {
				s.config.Logger.Info("Reconnection not successful")
				return
			}
		}
	}
}

// reconnect attempts to reconnec to RabbitMQ within configured
// maxReconnectionCount. If unsuccessful fatalError is triggered and false is
// returned. If successful true is returned and connection is reestablished.
func (s *Worker) reconnect(reason *amqp.Error) bool {
	var reconnectCount int
	for reconnectCount = 0; reconnectCount < s.config.MaxReconnectionAttempts; reconnectCount++ {
		s.config.Logger.Infof("Reconnecting to RabbitMQ after connection closed: attempt %d of %d: %v", reconnectCount+1, s.config.MaxReconnectionAttempts, reason)
		err := s.connect()
		if err != nil {
			s.config.Logger.Infof("Failed to reconnect to RabbitMQ: %v", err)
			time.Sleep(s.config.ReconnectionTimeout)
			continue
		}
		s.config.Logger.Info("Successfully reconnected to RabbitMQ")
		return true
	}
	reason.Reason = fmt.Sprintf("Tried to reconnect %d times. Giving up: %s", reconnectCount, reason.Reason)
	s.fatalError <- reason
	return false
}

func declareExchange(channel *amqp.Channel, exchangeName string) error {
	return channel.ExchangeDeclare(
		exchangeName,
		"topic", // kind
		true,    // durable
		false,   // autoDelete
		false,   // internal
		false,   // noWait
		nil,     // args
	)
}
