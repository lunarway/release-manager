package apis

import (
	"context"
	"net/http"
)

type FakeExporter struct {
	Sent []Message
}

func (f *FakeExporter) Send(_ context.Context, _ *http.Client, message Message) error {
	f.Sent = append(f.Sent, message)
	return nil
}
