package rabbitmq

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type publisher struct {
	channel      *amqp.Channel
	exchangeName string
	routingKey   string
}

func newPublisher(amqpConn *amqp.Connection, exchangeName string) (*publisher, error) {
	channel, err := amqpConn.Channel()
	if err != nil {
		return nil, errors.WithMessage(err, "get channel")
	}
	err = declareExchange(channel, exchangeName)
	if err != nil {
		return nil, errors.WithMessage(err, "declare exchange")
	}
	return &publisher{
		channel:      channel,
		exchangeName: exchangeName,
	}, nil
}

func (p *publisher) Publish(messageID string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return errors.WithMessage(err, "marshal message")
	}
	pub := amqp.Publishing{
		ContentType: "application/json",
		Body:        data,
		MessageId:   messageID,
	}
	// TODO: add publisher confirms to the channel and wait for acknowledgement
	// before returning
	err = p.channel.Publish(p.exchangeName, p.routingKey, false, false, pub)
	if err != nil {
		return err
	}
	return nil
}

func (p *publisher) Close() error {
	err := p.channel.Close()
	if err != nil {
		return errors.WithMessage(err, "close amqp channel")
	}
	return nil
}
