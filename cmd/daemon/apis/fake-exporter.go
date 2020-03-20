package apis

import (
	"context"
	"fmt"
	"net/http"
)

type FakeExporter struct {
	Sent []Message
}

func (f *FakeExporter) Send(_ context.Context, _ *http.Client, message Message) error {
	f.Sent = append(f.Sent, message)
	return nil
}

func (f *FakeExporter) NewLine() string {
	return "\n"
}

func (f *FakeExporter) FormatLink(link string, name string) string {
	return fmt.Sprintf("<%s|%s>", link, name)
}

func (f *FakeExporter) Name() string {
	return "Fake"
}
