package webhook

import (
	"encoding/json"

	"github.com/lunarway/release-manager/internal/broker"
	"github.com/lunarway/release-manager/internal/broker/amqpextra"
	"gopkg.in/go-playground/webhooks.v5/github"
)

var _ broker.Publishable = Payload{}

func NewPayload(p github.PushPayload) Payload {
	return Payload{
		Payload: p,
	}
}

type Payload struct {
	Payload github.PushPayload
}

func (p Payload) Type() string {
	return p.Type()
}

func (p Payload) Marshal() ([]byte, error) {
	return json.Marshal(p.Payload)
}

func (p Payload) Unmarshal(b []byte) error {
	var payload github.PushPayload
	return json.Unmarshal(b, &payload)
}

func (p Payload) Exchange() string {
	return amqpextra.ExchangeFanout
}
func (p Payload) RoutingKey() string {
	return "" // Fanout exchanges does not utilize the routingkey so we default to the empty string.
}
