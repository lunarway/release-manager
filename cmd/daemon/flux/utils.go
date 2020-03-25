package flux

import (
	"encoding/json"
	"io"

	"github.com/weaveworks/flux/event"
)

// ParseFluxEvent for doing flux event from Json into a flux Event struct.
func ParseFluxEvent(reader io.Reader) (event.Event, error) {
	var evt event.Event
	err := json.NewDecoder(reader).Decode(&evt)
	return evt, err
}
