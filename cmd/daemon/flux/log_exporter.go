package flux

import (
	"context"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/weaveworks/flux/event"
)

type LogExporter struct {
	Log *log.Logger
}

func (f *LogExporter) Send(_ context.Context, message Message) error {
	f.Log.With("FluxMessage", message).Infof("flux message exported")
	return nil
}

type Message struct {
	TitleLink string
	Body      string
	Type      string
	Title     string
	Event     event.Event
}
