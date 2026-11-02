package events

import (
	"encoding/json"

	"github.com/lunarway/release-manager/internal/broker"
)

// ConfigChangedEvent notifies all replicas that the config repository changed.
// It carries the new master HEAD SHA so receivers can skip syncing when their
// local clone is already current.
type ConfigChangedEvent struct {
	SHA string `json:"sha,omitempty"`
}

// Marshal implements broker.Publishable.
func (e ConfigChangedEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// Unmarshal implements broker.Publishable.
func (e *ConfigChangedEvent) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

var _ broker.Publishable = &ConfigChangedEvent{}

// Type implements broker.Publishable.
func (e ConfigChangedEvent) Type() string {
	return "config_changed"
}
