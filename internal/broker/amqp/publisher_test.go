package amqp

import (
	"context"
	"testing"
	"time"

	"github.com/lunarway/release-manager/internal/broker"
	"github.com/lunarway/release-manager/internal/test"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

func TestRawPublisher_Publish(t *testing.T) {
	rabbitHost := test.RabbitMQIntegration(t)
	tt := []struct {
		name   string
		action func(p *rawPublisher) error
		err    error
	}{
		{
			name: "no error conditions",
			action: func(p *rawPublisher) error {
				return p.Publish(context.Background(), "testEvent", "event-id", []byte(`hello world`))
			},
			err: nil,
		},
		{
			name: "context cancellation",
			action: func(p *rawPublisher) error {
				err := p.channel.Close()
				if err != nil {
					return err
				}
				ctx, done := context.WithCancel(context.Background())
				go func() {
					time.Sleep(500 * time.Millisecond)
					done()
				}()
				return p.Publish(ctx, "testEvent", "event-id", []byte(`hello world`))
			},
			err: context.Canceled,
		},
		{
			name: "close publisher",
			action: func(p *rawPublisher) error {
				err := p.channel.Close()
				if err != nil {
					return err
				}
				go func() {
					time.Sleep(500 * time.Millisecond)
					p.Close()
				}()
				return p.Publish(context.Background(), "testEvent", "event-id", []byte(`hello world`))
			},
			err: broker.ErrBrokerClosed,
		},
		{
			// ensures we don't panic on the close channel so no error is returned nor
			// expected
			name: "close publisher multiple times",
			action: func(p *rawPublisher) error {
				p.Close()
				p.Close()
				return nil
			},
			err: nil,
		},
		{
			name: "publish after close",
			action: func(p *rawPublisher) error {
				p.Close()
				return p.Publish(context.Background(), "testEvent", "event-id", []byte(`hello world`))
			},
			err: broker.ErrBrokerClosed,
		},
		// TODO: implement handling of lost channels to avoid stuck publishing
		// {
		// 	name: "close channel",
		// 	action: func(p *rawPublisher) error {
		// 		err := p.channel.Close()
		// 		if err != nil {
		// 			return err
		// 		}
		// 		return p.Publish(context.Background(), "testEvent", "event-id", []byte(`hello world`))
		// 	},
		// 	err: nil,
		// },
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			c := ConnectionConfig{
				Host:        rabbitHost,
				User:        "lunar",
				Password:    "lunar",
				VirtualHost: "/",
				Port:        5672,
			}
			amqpConn, err := amqp.Dial(c.Raw())
			if !assert.NoError(t, err, "unexpected dial error") {
				return
			}
			defer amqpConn.Close()

			p, err := newPublisher(amqpConn, "", 100*time.Millisecond, func(ctx context.Context, reason error) {
				t.Helper()
				t.Logf("Republishing: %v", reason)
			})
			if !assert.NoError(t, err, "unexpected publisher instantiation error") {
				return
			}
			defer p.Close()

			err = tc.action(p)

			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error(), "error is not as expected")
			} else {
				assert.NoError(t, err, "unexpected error")
			}
		})
	}
}
