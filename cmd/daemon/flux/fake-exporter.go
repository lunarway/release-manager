package flux

import (
	"context"
)

type FakeExporter struct {
	Sent []Message
}

func (f *FakeExporter) Send(_ context.Context, message Message) error {
	f.Sent = append(f.Sent, message)
	return nil
}
