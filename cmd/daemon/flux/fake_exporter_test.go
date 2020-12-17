package flux_test

import (
	"context"

	"github.com/lunarway/release-manager/internal/flux"
)

type FakeExporter struct {
	Sent []flux.Event
}

func (f *FakeExporter) Send(_ context.Context, event flux.Event) error {
	f.Sent = append(f.Sent, event)
	return nil
}
