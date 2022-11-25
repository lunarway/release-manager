package events

import (
	"encoding/json"

	"github.com/lunarway/release-manager/internal/broker"
)

type Envelope struct {
	EventName string          `json:"eventName,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
}

// Marshal implements broker.Publishable
func (re Envelope) Marshal() ([]byte, error) {
	return json.Marshal(re)
}

// Unmarshal implements broker.Publishable
func (re Envelope) Unmarshal(data []byte) error {
	return json.Unmarshal(data, &re)
}

var _ broker.Publishable = &Envelope{}

func (re Envelope) Type() string {
	return re.EventName
}
