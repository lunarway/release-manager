package flux

import (
	"context"
	"encoding/json"
	"io"

	"github.com/weaveworks/flux/event"
)

type Message struct {
	TitleLink string
	Body      string
	Type      string
	Title     string
	Event     event.Event
}

// An exporter sends a formatted event to an upstream.
type Exporter interface {
	// Send a message through the exporter.
	Send(c context.Context, message Message) error
}

// ParseFluxEvent for doing flux event from Json into a flux Event struct.
func ParseFluxEvent(reader io.Reader) (event.Event, error) {
	var evt event.Event
	err := json.NewDecoder(reader).Decode(&evt)
	return evt, err
}
