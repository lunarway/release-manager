package flux_test

import (
	"context"

	"github.com/lunarway/release-manager/cmd/daemon/flux"
)

type FakeExporter struct {
	Sent []flux.Message
}

func (f *FakeExporter) Send(_ context.Context, message flux.Message) error {
	f.Sent = append(f.Sent, message)
	return nil
}
