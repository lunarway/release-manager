package flux_test

import (
	"context"

	fluxevent "github.com/weaveworks/flux/event"
)

type FakeExporter struct {
	Sent []fluxevent.Event
}

func (f *FakeExporter) Send(_ context.Context, event fluxevent.Event) error {
	f.Sent = append(f.Sent, event)
	return nil
}
