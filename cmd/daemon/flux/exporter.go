package flux

import (
	"context"
	"net/http"

	httpinternal "github.com/lunarway/release-manager/internal/http"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/weaveworks/flux/event"
	"github.com/weaveworks/flux/update"
)

type Message struct {
	Event event.Event
}

// Exporter sends a formatted event to an upstream.
type Exporter interface {
	// Send a message through the exporter.
	Send(c context.Context, event event.Event) error
}
type ReleaseManagerExporter struct {
	Log         *log.Logger
	Environment string
	Client      httpinternal.Client
}

func (f *ReleaseManagerExporter) Send(_ context.Context, event event.Event) error {
	f.Log.With("event", event).Infof("flux event logged")
	var resp httpinternal.FluxNotifyResponse
	url, err := f.Client.URL("webhook/daemon/flux")
	if err != nil {
		return err
	}
	err = f.Client.Do(http.MethodPost, url, httpinternal.FluxNotifyRequest{
		Environment:        f.Environment,
		EventID:            event.ID,
		EventServiceIDs:    event.ServiceIDs,
		EventChangedImages: getChangedImages(event.Metadata),
		EventResult:        getResult(event.Metadata),
		EventType:          event.Type,
		EventStartedAt:     event.StartedAt,
		EventEndedAt:       event.EndedAt,
		EventLogLevel:      event.LogLevel,
		EventMessage:       event.Message,
		EventString:        event.String(),
		Commits:            getCommits(event.Metadata),
		Errors:             getErrors(event.Metadata),
	}, &resp)
	if err != nil {
		return err
	}
	return nil
}

func getCommits(meta event.EventMetadata) []event.Commit {
	switch v := meta.(type) {
	case *event.CommitEventMetadata:
		return []event.Commit{
			event.Commit{
				Revision: v.Revision,
			},
		}
	case *event.SyncEventMetadata:
		return v.Commits
	default:
		return []event.Commit{}
	}
}

func getResult(meta event.EventMetadata) update.Result {
	switch v := meta.(type) {
	case *event.AutoReleaseEventMetadata:
		return v.Result
	case *event.ReleaseEventMetadata:
		return v.Result
	default:
		return update.Result{}
	}
}

func getChangedImages(meta event.EventMetadata) []string {
	switch v := meta.(type) {
	case *event.AutoReleaseEventMetadata:
		return v.Result.ChangedImages()
	case *event.ReleaseEventMetadata:
		return v.Result.ChangedImages()
	default:
		return []string{}
	}
}

func getErrors(meta event.EventMetadata) []event.ResourceError {
	switch v := meta.(type) {
	case *event.SyncEventMetadata:
		return v.Errors
	default:
		return []event.ResourceError{}
	}
}
