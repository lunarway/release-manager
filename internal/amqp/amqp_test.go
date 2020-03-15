package amqp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/test"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

type testEvent struct {
	Message string `json:"message"`
}

func (testEvent) Type() string {
	return "test-event"
}

func (t testEvent) Body() interface{} {
	return t
}

// TestWorker_PublishAndConsumer tests that we can publish and receive messages
// with a worker.
func TestWorker_PublishAndConsumer(t *testing.T) {
	rabbitHost := test.RabbitMQIntegration(t)
	epoch := time.Now().UnixNano()
	exchange := fmt.Sprintf("rm-test-exchange-%d", epoch)
	queue := fmt.Sprintf("rm-test-queue-%d", epoch)
	logger := log.New(&log.Configuration{
		Level: log.Level{
			Level: zapcore.DebugLevel,
		},
		Development: true,
	})

	publishedMessages := 100
	var receivedCount int32
	receivedAllEvents := make(chan struct{})
	worker, err := NewWorker(Config{
		Connection: ConnectionConfig{
			Host:        rabbitHost,
			User:        "lunar",
			Password:    "lunar",
			VirtualHost: "/",
			Port:        5672,
		},
		MaxReconnectionAttempts: 0,
		ReconnectionTimeout:     50 * time.Millisecond,
		Exchange:                exchange,
		Queue:                   queue,
		RoutingKey:              "#",
		Prefetch:                10,
		Logger:                  logger,
		Handlers: map[string]func(d []byte) error{
			testEvent{}.Type(): func(d []byte) error {
				newCount := atomic.AddInt32(&receivedCount, 1)
				if int(newCount) == publishedMessages {
					close(receivedAllEvents)
				}
				var msg testEvent
				err := json.Unmarshal(d, &msg)
				if err != nil {
					return err
				}
				logger.Infof("Received %s", msg.Message)
				return nil
			},
		},
	})
	if !assert.NoError(t, err, "unexpected init error") {
		return
	}

	var consumerWg sync.WaitGroup
	consumerWg.Add(1)
	go func() {
		defer consumerWg.Done()
		err := worker.StartConsumer()
		if err != ErrWorkerClosed {
			assert.NoError(t, err, "unexpected consumer error")
		}
	}()

	var publisherWg sync.WaitGroup
	publisherWg.Add(1)
	go func() {
		logger.Infof("TEST: Starting to publish %d messages", publishedMessages)
		defer publisherWg.Done()
		for i := 1; i <= publishedMessages; i++ {
			logger.Infof("TEST: Published message %d", i)
			err := worker.Publish(context.Background(), testEvent{
				Message: fmt.Sprintf("Message %d", i),
			})
			assert.NoError(t, err, "unexpected error publishing message")
		}
	}()
	// block until all messages are sent
	publisherWg.Wait()

	// block until all messages are received with a timeout
	select {
	case <-receivedAllEvents:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting to receive all events")
	}

	err = worker.Close()
	assert.NoError(t, err, "unexpected close error")

	// wait for consumer Go routine to exit
	consumerWg.Wait()
	assert.Equal(t, publishedMessages, int(receivedCount), "received messages count not as expected")
}

// TestWorker_StartConsumer_reconnection tests the reconnection mechanism of the
// worker ensuring that network failures and alike are handled accordingly.
func TestWorker_StartConsumer_reconnection(t *testing.T) {
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
		workerError             string
	}{
		{
			name:                    "zero reconnection attempts",
			maxReconnectionAttempts: 0,
			maxSuccessfullDials:     1,
			workerError:             "Tried to reconnect 0 times. Giving up",
		},
		{
			name:                    "no recovery",
			maxReconnectionAttempts: 2,
			maxSuccessfullDials:     1,
			workerError:             "Tried to reconnect 2 times. Giving up",
		},
		{
			name:                    "recovery after a dial error",
			maxReconnectionAttempts: 2,
			maxSuccessfullDials:     2,
			workerError:             ErrWorkerClosed.Error(),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			worker, err := NewWorker(Config{
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
				RoutingKey:              "#",
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
				// sleep for a short amount of time to let the worker.Start method be
				// called outside to Go routine
				time.Sleep(10 * time.Millisecond)

				logger.Infof("TEST: Killing TCP connection to RabbitMQ")
				err = conn.Close()
				if err != nil {
					t.Errorf("close net.Conn to AMQP failed: %v", err)
					return
				}

				// wait some seconds before shutting down for retries to take place
				time.Sleep(200 * time.Millisecond)
				logger.Infof("TEST: Shutting down worker...")

				// shutdown rabbit connection
				err = worker.Close()
				assert.NoError(t, err, "unexpected close error")
			}()

			// this is the functional assertion. We expect Start to block until as
			// long as we are able to keep a connection open with retries. If we
			// have no more retry attempts left the function will return with an
			// error.
			err = worker.StartConsumer()
			logger.Infof("TEST: worker error: %v", err)
			if assert.Error(t, err, "expected a worker error") {
				assert.Regexp(t, tc.workerError, err.Error(), "error string not as expected")
			} else {
				// the worker should always return an error when terminating.
				t.Fatal("expected a worker error but received none")
			}
			wg.Wait()
		})
	}
}
