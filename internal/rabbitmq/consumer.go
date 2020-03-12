package rabbitmq

import (
	"fmt"
	"time"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

// Consumer is a RabbitMQ consumer. It will setup an AMQP channel to consume
// messages from an exchange through a queue and will withstand disconnects on
// connection to RabbitMQ.
//
// Reconnection is implemented as a chan *connection that is consumed in the
// Start method. If connection loss is detected the reconnector Go routine will
// setup a new connection and push it on to the channel thus keeping Start
// blocking.
type Consumer struct {
	config Config

	// currentConnection provides an active connection to consume from.
	currentConnection chan *connection
	// shutdown is used to terminate the different Go routines in the consumer. It
	// will be closed as a signal to stop.
	shutdown chan struct{}
	// connectionClosed is used to signal that a connection was lost and that the
	// reconnector should attempt to reestablish it.
	connectionClosed chan *amqp.Error
	// fatalError is used to signal that the consumer was unable to recreate a
	// connection and that the consumer should stop consuming messages.
	fatalError chan error
}

type Config struct {
	Connection              ConnectionConfig
	Exchange                string
	Queue                   string
	MaxReconnectionAttempts int
	ReconnectionTimeout     time.Duration
	Handler                 func(amqp.Delivery) error
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

// New allocates and returns a Consumer of messages from a RabbitMQ exchange.
func New(c Config) (*Consumer, error) {
	consumer := Consumer{
		shutdown:          make(chan struct{}),
		currentConnection: make(chan *connection),
		fatalError:        make(chan error),
		connectionClosed:  make(chan *amqp.Error),
		config:            c,
	}
	consumer.config.Logger = c.Logger.WithFields(
		"host", c.Connection.Host,
		"user", c.Connection.User,
		"port", c.Connection.Port,
		"virtualHost", c.Connection.VirtualHost,
		"exchange", c.Exchange,
		"queue", c.Queue,
		"maxReconnectionAttempts", c.MaxReconnectionAttempts,
		"reconnectionTimeout", c.ReconnectionTimeout,
		"amqpConfig", fmt.Sprintf("%#v", c.AMQPConfig),
	)

	err := consumer.connect()
	if err != nil {
		return nil, err
	}
	go consumer.reconnector()
	return &consumer, nil
}

func (s *Consumer) Close() error {
	close(s.shutdown)
	return nil
}

// ErrConsumerClosed is returned by the Consumer's Start method after a call to
// Close.
var ErrConsumerClosed = errors.New("rabbitmq: Consumer closed")

// Start starts the consumption of messages from the queue. The method is
// blocking and will only return if the consumer is stopped with Close or the
// connection is lost and cannot be recreated.
func (s *Consumer) Start() error {

	for {
		select {
		// consumer is instructed to shutdown from a Close call
		case <-s.shutdown:
			return ErrConsumerClosed

		// consumer is unable to recover a network failure and should stop
		case err := <-s.fatalError:
			return err

		// consumer has a new connection that can be used to consume messages with the handler
		case conn := <-s.currentConnection:
			// this call is blocking as long as the connection is available.
			err := conn.Start(s.config.Handler)
			if err != nil {
				return err
			}
		}
	}
}

func (s *Consumer) connect() error {
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
	conn := connection{
		amqpConnection:           amqpConn,
		connectionClosedListener: make(chan *amqp.Error),
		logger:                   c.Logger,
	}
	conn.amqpConnection.NotifyClose(conn.connectionClosedListener)
	err = conn.Init(c.Exchange, c.Queue)
	if err != nil {
		return errors.WithMessage(err, "initialize connection")
	}

	// listen for connection failures on the specific connection along with
	// closing the connection if general shutdown is signalled
	go func() {
		c.Logger.Info("Connection close listener started")
		defer c.Logger.Info("Connection close listener stopped")
		select {
		case <-s.shutdown:
			err := conn.Close()
			if err != nil {
				c.Logger.Errorf("Failed to close amqp connection: %v", err)
			}
		case err, abnormalShutdown := <-conn.connectionClosedListener:
			if !abnormalShutdown {
				c.Logger.Info("Connection closed due to normal shutdown")
				return
			}
			c.Logger.Info("Connection closed due to abnormal shutdown")
			// signal the consumer that the connection was lost
			s.connectionClosed <- err
		}
	}()
	go func() {
		select {
		case <-s.shutdown:
		case s.currentConnection <- &conn:
		}
	}()
	c.Logger.Info("Connected to RabbitMQ successfully")
	return nil
}

func (s *Consumer) reconnector() {
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

// reconnect attempts to reconnecto to RabbitMQ within configured
// maxReconnectionCount. If unsuccessful fatalError is triggered and false is
// returned. If successful true is returned and connection is reestablished.
func (s *Consumer) reconnect(reason *amqp.Error) bool {
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

type connection struct {
	amqpConnection           *amqp.Connection
	connectionClosedListener chan *amqp.Error
	channel                  *amqp.Channel
	queue                    *amqp.Queue
	logger                   *log.Logger
}

func (c *connection) Init(exchange, queue string) error {
	channel, err := c.amqpConnection.Channel()
	if err != nil {
		return errors.WithMessage(err, "get channel")
	}
	c.channel = channel
	routingKey := "#"

	err = c.channel.ExchangeDeclare(exchange, // name
		"topic", // kind
		true,    // durable
		false,   // autoDelete
		false,   // internal
		false,   // noWait
		nil,     // args
	)
	if err != nil {
		return errors.WithMessage(err, "declare exchange")
	}

	q, err := c.channel.QueueDeclare(
		queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return errors.WithMessage(err, "declare queue")
	}
	c.queue = &q
	err = c.channel.QueueBind(queue, routingKey, exchange, false, nil)
	if err != nil {
		return errors.WithMessage(err, "bind to queue")
	}
	return nil
}

func (c *connection) Close() error {
	if c.amqpConnection.IsClosed() {
		return nil
	}
	// TODO: stop consuming messages with c.channel.Cancel() and wait for the
	// consumer to complete before closing the connection to avoid dropping
	// inflight messages.
	err := c.amqpConnection.Close()
	if err != nil {
		return errors.WithMessage(err, "close amqp connection")
	}
	return nil
}

func (c *connection) Start(handler func(amqp.Delivery) error) error {
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
		logger := c.logger.WithFields(
			"exchange", msg.Exchange,
			"routingKey", msg.RoutingKey,
			"messageId", msg.MessageId,
			"timestamp", msg.Timestamp,
			"headers", fmt.Sprintf("%#v", msg.Headers),
		)
		logger.Infof("Received message from exchange=%s routingKey=%s messageId=%s timestamp=%s", msg.Exchange, msg.RoutingKey, msg.MessageId, msg.Timestamp)
		err := handler(msg)
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
