package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lunarway/release-manager/internal/broker"
	"github.com/lunarway/release-manager/internal/log"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

var _ broker.Broker = &Broker{}
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

// TestBroker_PublishAndConsumer tests that we can publish and receive messages
// with a Broker.
func TestBroker_PublishAndConsumer(t *testing.T) {
	logger := log.New(&log.Configuration{
		Level: log.Level{
			Level: zapcore.DebugLevel,
		},
		Development: true,
	})

	publishedMessages := 100
	var receivedCount int32
	receivedAllEvents := make(chan struct{})
	memoryBroker := New(logger, 5)

	var consumerWg sync.WaitGroup
	consumerWg.Add(1)
	go func() {
		defer consumerWg.Done()
		err := memoryBroker.StartConsumer(map[string]func([]byte) error{
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
			err := memoryBroker.Publish(context.Background(), &testEvent{
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

	err := memoryBroker.Close()
	assert.NoError(t, err, "unexpected close error")

	// wait for consumer Go routine to exit
	consumerWg.Wait()
	assert.Equal(t, publishedMessages, int(receivedCount), "received messages count not as expected")
}
