package flux

import (
	"context"
	"fmt"
	"net/http"

	"github.com/lunarway/release-manager/internal/flux"
	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/lunarway/release-manager/internal/log"
)

type Message struct {
	Event flux.Event
}

// Exporter sends a formatted event to an upstream.
type Exporter interface {
	// Send a message through the exporter.
	Send(c context.Context, event flux.Event) error
}
type ReleaseManagerExporter struct {
	Log         *log.Logger
	Environment string
	Client      httpinternal.Client
}

func (f *ReleaseManagerExporter) Send(_ context.Context, event flux.Event) error {
	f.Log.With("event", fmt.Sprintf("%#v", event)).Infof("flux event logged")
	var resp httpinternal.FluxNotifyResponse
	url, err := f.Client.URL("webhook/daemon/flux")
	if err != nil {
		return err
	}
	err = f.Client.Do(http.MethodPost, url, httpinternal.FluxNotifyRequest{
		Environment: f.Environment,
		FluxEvent:   event,
	}, &resp)
	if err != nil {
		return err
	}
	return nil
}
