package flux

import (
	"context"
	"encoding/json"
	"io"

	fluxevent "github.com/weaveworks/flux/event"
)

type Message struct {
	TitleLink string
	Body      string
	Type      string
	Title     string
	Event     fluxevent.Event
}

// An exporter sends a formatted event to an upstream.
type Exporter interface {
	// Send a message through the exporter.
	Send(c context.Context, message Message) error
}

// ParseFluxEvent for doing flux event from Json into a flux Event struct.
func ParseFluxEvent(reader io.Reader) (fluxevent.Event, error) {
	var event fluxevent.Event
	err := json.NewDecoder(reader).Decode(&event)
	return event, err
}
