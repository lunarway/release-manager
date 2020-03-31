package flux_test

import (
	"context"

	"github.com/weaveworks/flux/event"
)

type FakeExporter struct {
	Sent []event.Event
}

func (f *FakeExporter) Send(_ context.Context, event event.Event) error {
	f.Sent = append(f.Sent, event)
	return nil
}
