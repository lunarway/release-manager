package amqp

import (
	"context"
	"fmt"
	"time"

	"github.com/makasim/amqpextra"
	"github.com/makasim/amqpextra/publisher"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Worker is an AMQP consumer and publisher implementation. It configures an
// AMQP channel to consume messages from a queue along with a publisher to
// publish messages to an exchange.
//
// It transparently recovers from connection failures and publishes are blocked
// during recovery.
type Worker struct {
	logger Logger
	prefix string

	// ctxCancel is used to signal dialer, consumers and publishers to start
	// closing down
	ctxCancel         func()
	dialer            *amqpextra.Dialer
	publisher         *publisher.Publisher
	declaredExchanges map[string]struct{}

	PublishMode string
}

// Logger is the logging interface required by a Worker.
type Logger interface {
	Debugf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
	Infof(template string, args ...interface{})
}

// Config contains configuration parameters for a Worker.
type Config struct {
	Logger                          Logger
	ConnectionString                string
	VirtualHost                     string
	Prefix                          string
	ReconnectionTimeout             time.Duration
	ConnectTimeout                  time.Duration
	InitTimeout                     time.Duration
	MaxUnconfirmedInFlightPublishes uint
	OnDial                          func(attempt int, err error)
}

// New allocates, initializes and returns a new Worker instance.
func New(c Config) (*Worker, error) {
	// this context is used to signal the dialer, child consumers and publishers
	// to stop by calling its cancel func.
	// Observe that this context is very long running, it's alive for as long as the worker is alive and is used
	// to signal the worker to shutdown via its cancel func. Hence, we cannot use this context to implement initial
	// timout on establishing the connection
	ctx, cancel := context.WithCancel(context.Background())
	dialAttempt := 0
	dialerState := make(chan amqpextra.State, 2)
	dialer, err := amqpextra.NewDialer(
		// FIXME: we cannot set Vhost or Heartbeat (defaults to 30s) with an Option
		amqpextra.WithURL(c.ConnectionString),
		amqpextra.WithRetryPeriod(c.ReconnectionTimeout),
		amqpextra.WithLogger(newLogger(c.Logger, "dialer")),
		amqpextra.WithContext(ctx),
		amqpextra.WithNotify(dialerState),
		amqpextra.WithAMQPDial(func(url string, amqpConf amqp.Config) (connection amqpextra.AMQPConnection, err error) {
			amqpConf.Dial = amqp.DefaultDial(c.ConnectTimeout)
			//TODO if we want to, we can actually set the Heartbeat here:
			//amqpConf.Heartbeat = c.Heartbeat
			dialAttempt++
			conn, connErr := amqp.DialConfig(url, amqpConf)
			if c.OnDial != nil {
				c.OnDial(dialAttempt, connErr)
			}
			return conn, connErr
		}),
	)
	if err != nil {
		cancel()
		return nil, errors.WithMessage(err, "instantiate dialer")
	}

	err = waitForDialerReady(dialerState, c.InitTimeout)
	if err != nil {
		cancel()
		return nil, err
	}

	publisherState := make(chan publisher.State, 2)
	publisherOptions := []publisher.Option{
		publisher.WithLogger(newLogger(c.Logger, "publisher")),
		publisher.WithNotify(publisherState),
	}
	var publishMode string
	if c.MaxUnconfirmedInFlightPublishes != 0 {
		c.Logger.Infof("[amqp] Confirmation publisher mode enabled with max %d unconfirmed in-flight publishes", c.MaxUnconfirmedInFlightPublishes)
		publisherOptions = append(publisherOptions, publisher.WithConfirmation(c.MaxUnconfirmedInFlightPublishes))
		publishMode = "confirm"
	} else {
		c.Logger.Infof("[amqp] Transactional publisher mode enabled")
		publishMode = "transactional"
	}
	publisher, err := dialer.Publisher(publisherOptions...)
	if err != nil {
		cancel()
		return nil, errors.WithMessage(err, "instantiate publisher")
	}

	err = waitForPublisherReady(publisherState, c.InitTimeout)
	if err != nil {
		cancel()
		return nil, err
	}

	w := &Worker{
		dialer:    dialer,
		logger:    c.Logger,
		prefix:    c.Prefix,
		ctxCancel: cancel,

		publisher:         publisher,
		PublishMode:       publishMode,
		declaredExchanges: make(map[string]struct{}),
	}
	return w, nil
}

func (w *Worker) prefixed(s string) string {
	return w.prefix + s
}

// Close closes the worker stopping any active consumers and publishes.
func (w *Worker) Close() error {
	w.logger.Debugf("[amqp] Closing AMQP worker")
	w.ctxCancel()

	w.logger.Debugf("[amqp] Waiting for AMQP publisher to close")
	<-w.publisher.NotifyClosed()

	w.logger.Debugf("[amqp] Waiting for AMQP dialer to close")
	<-w.dialer.NotifyClosed()

	return nil
}

func waitForDialerReady(dialerState chan amqpextra.State, timeout time.Duration) error {
	err := waitForReady(func(ready chan struct{}) {
		state := <-dialerState
		if state.Ready != nil {
			close(ready)
		}
	}, timeout)
	if err != nil {
		return fmt.Errorf("amqp dialer: %w", err)
	}
	return nil
}

func waitForPublisherReady(publisherState chan publisher.State, timeout time.Duration) error {
	err := waitForReady(func(ready chan struct{}) {
		state := <-publisherState
		if state.Ready != nil {
			close(ready)
		}
	}, timeout)
	if err != nil {
		return fmt.Errorf("amqp publisher: %w", err)
	}
	return nil
}

// waitForReady blocks until the ready function closes the ready channel or the
// timeout expires.
func waitForReady(ready func(chan struct{}), timeout time.Duration) error {
	timeoutExpired := time.After(timeout)
	readyResult := make(chan struct{})

	ready(readyResult)

	// wait for the ready function to report a result
	for {
		select {
		case <-readyResult:
			return nil
		case <-timeoutExpired:
			return fmt.Errorf("not ready after %v", timeout)
		}
	}
}
