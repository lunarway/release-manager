package amqp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lunarway/release-manager/internal/broker"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/lunarway/release-manager/internal/test"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

var _ broker.Broker = &Worker{}
var _ broker.Publishable = &testEvent{}

type testEvent struct {
	Message string `json:"message"`
}

func (testEvent) Type() string {
	return "test-event"
}

func (t testEvent) Marshal() ([]byte, error) {
	return json.Marshal(t)
}

func (t *testEvent) Unmarshal(d []byte) error {
	return json.Unmarshal(d, t)
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
		ReconnectionTimeout: 50 * time.Millisecond,
		Exchange:            exchange,
		Queue:               queue,
		RoutingKey:          "#",
		Prefetch:            10,
		Logger:              logger,
	})
	if !assert.NoError(t, err, "unexpected init error") {
		return
	}

	var consumerWg sync.WaitGroup
	consumerWg.Add(1)
	go func() {
		defer consumerWg.Done()
		err := worker.StartConsumer(map[string]func([]byte) error{
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
		})
		assert.EqualError(t, err, broker.ErrBrokerClosed.Error(), "unexpected consumer error")
	}()

	var publisherWg sync.WaitGroup
	publisherWg.Add(1)
	go func() {
		logger.Infof("TEST: Starting to publish %d messages", publishedMessages)
		defer publisherWg.Done()
		for i := 1; i <= publishedMessages; i++ {
			logger.Infof("TEST: Published message %d", i)
			err := worker.Publish(context.Background(), &testEvent{
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

// TestWorker_reconnection tests the reconnection mechanism of the worker
// ensuring that network failures and alike are mitigated by reconnecting.
func TestWorker_reconnection(t *testing.T) {
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
	dialFunc := func() func(string, string) (net.Conn, error) {
		dials := 1
		return func(network, addr string) (net.Conn, error) {
			logger.Infof("Dialling attempt %d", dials)
			defer func() {
				dials = dials + 1
			}()
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

	epoch := time.Now().UnixNano()
	var consumedCount int32
	worker, err := NewWorker(Config{
		Connection: ConnectionConfig{

			Host:        rabbitHost,
			User:        "lunar",
			Password:    "lunar",
			VirtualHost: "/",
			Port:        5672,
		},
		ReconnectionTimeout: 1 * time.Millisecond,
		Exchange:            fmt.Sprintf("%s_%d", t.Name(), epoch),
		Queue:               fmt.Sprintf("%s_%d", t.Name(), epoch),
		RoutingKey:          "#",
		AMQPConfig: &amqp.Config{
			Dial:  dialFunc(),
			Vhost: "/",
		},
		Logger: logger,
	})
	if !assert.NoError(t, err, "unexpected init error") {
		return
	}

	// setup a go routine that will publish a message, kill the connection and
	// publish a new message
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// sleep for a short amount of time to let the worker.Start method be
		// called outside to Go routine
		time.Sleep(10 * time.Millisecond)

		err := worker.Publish(context.Background(), &testEvent{
			Message: "message 1",
		})
		if err != nil {
			t.Errorf("publish message 1: %v", err)
		}

		logger.Infof("TEST: Killing TCP connection to RabbitMQ")
		err = conn.Close()
		if err != nil {
			t.Errorf("close net.Conn to AMQP failed: %v", err)
		}

		// wait some seconds before publishing again for retries to take place
		time.Sleep(100 * time.Millisecond)

		err = worker.Publish(context.Background(), &testEvent{
			Message: "message 2",
		})
		if err != nil {
			t.Errorf("publish message 2: %v", err)
		}

		time.Sleep(100 * time.Millisecond)
		logger.Infof("TEST: Shutting down worker...")
		// shutdown rabbit connection
		err = worker.Close()
		assert.NoError(t, err, "unexpected close error")
	}()

	// this is the functional assertion. We expect Start to block until as
	// long as we are able to keep a connection open with retries. If we
	// have no more retry attempts left the function will return with an
	// error.
	err = worker.StartConsumer(map[string]func([]byte) error{
		testEvent{}.Type(): func(d []byte) error {
			logger.Infof("Handled %s", d)
			atomic.AddInt32(&consumedCount, 1)
			return nil
		},
	})
	logger.Infof("TEST: worker error: %v", err)
	assert.EqualError(t, err, broker.ErrBrokerClosed.Error(), "consumer returned unexpected error")
	wg.Wait()
	assert.Equal(t, int32(2), consumedCount, "did not receive two messages as exected")
}
