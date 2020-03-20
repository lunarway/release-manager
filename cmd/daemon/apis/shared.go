package apis

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

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
	Send(c context.Context, client *http.Client, message Message) error
}

// Interface for fetching configuration settings
type Config interface {
	// Get an optional setting by name, with a default value to return if it does
	// exist.
	Optional(key string, defaultValue string) string

	// Get a required setting by name.
	Required(key string) (string, error)
}

// Parse a flux event from Json into a flux Event struct.
func ParseFluxEvent(reader io.Reader) (event fluxevent.Event, err error) {
	err = json.NewDecoder(reader).Decode(&event)
	return
}
