package rabbitmq

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/test"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

// TestInit_reconnection tests the reconnection mechanism of the consumer
// ensuring that network failures and alike are handled accordingly.
func TestInit_reconnection(t *testing.T) {
	rabbitHost := test.RabbitMQIntegration(t)
	logger := log.New(&log.Configuration{
		Level: log.Level{
			Level: zapcore.DebugLevel,
		},
		Development: true,
	})

	var conn net.Conn
	// maxDials returns a dial function for amqp.Config configured to reject dials
	// after max attempts.
	maxDials := func(max int) func(string, string) (net.Conn, error) {
		dials := 1
		return func(network, addr string) (net.Conn, error) {
			logger.Infof("Dialling attempt %d", dials)
			defer func() {
				dials = dials + 1
			}()
			if dials > max {
				logger.Infof("Dialling attempt %d blocked", dials)
				return nil, errors.New("dial blocked")
			}
			var err error
			conn, err = net.DialTimeout(network, addr, 10*time.Second)
			if err != nil {
				return nil, err
			}
			err = conn.SetDeadline(time.Now().Add(10 * time.Second))
			if err != nil {
				return nil, err
			}
			return conn, nil
		}
	}
	tt := []struct {
		name                    string
		maxReconnectionAttempts int
		maxSuccessfullDials     int
		consumeError            string
	}{
		{
			name:                    "zero reconnection attempts",
			maxReconnectionAttempts: 0,
			maxSuccessfullDials:     1,
			consumeError:            "Tried to reconnect 0 times. Giving up",
		},
		{
			name:                    "no recovery",
			maxReconnectionAttempts: 2,
			maxSuccessfullDials:     1,
			consumeError:            "Tried to reconnect 2 times. Giving up",
		},
		{
			name:                    "recovery after a dial error",
			maxReconnectionAttempts: 2,
			maxSuccessfullDials:     2,
			consumeError:            ErrConsumerClosed.Error(),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			consumer, err := New(Config{
				Connection: ConnectionConfig{

					Host:        rabbitHost,
					User:        "lunar",
					Password:    "lunar",
					VirtualHost: "/",
					Port:        5672,
				},
				MaxReconnectionAttempts: tc.maxReconnectionAttempts,
				ReconnectionTimeout:     50 * time.Millisecond,
				Exchange:                "release-manager",
				Queue:                   "test-queue",
				AMQPConfig: &amqp.Config{
					Dial:  maxDials(tc.maxSuccessfullDials),
					Vhost: "/",
				},
				Logger: logger,
			})
			if !assert.NoError(t, err, "unexpected init error") {
				return
			}

			// setup a go routine that kills the connection after 1 second.
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				// sleep for a short amount of time to let the consumer get into a
				// consuming state
				time.Sleep(10 * time.Millisecond)

				logger.Infof("TEST: Killing TCP connection to RabbitMQ")
				err = conn.Close()
				if err != nil {
					t.Errorf("close net.Conn to AMQP failed: %v", err)
					return
				}

				// wait some seconds before shutting down for retries to take place
				time.Sleep(200 * time.Millisecond)
				logger.Infof("TEST: Shutting down consumer...")

				// shutdown rabbit connection
				err = consumer.Close()
				assert.NoError(t, err, "unexpected close error")
			}()

			// this is the functional assertion. We expect Start to block until as
			// long as we are able to keep a connection open with retries. If we
			// have no more retry attempts left the function will return with an
			// error.
			err = consumer.Start()
			logger.Infof("TEST: consumer error: %v", err)
			if assert.Error(t, err, "expected a consumer error") {
				assert.Regexp(t, tc.consumeError, err.Error(), "error string not as expected")
			} else {
				// the consumer should always return an error when terminating.
				t.Fatal("expected a consumer error but received none")
			}
			wg.Wait()
		})
	}
}
