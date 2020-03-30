package flux

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	httpinternal "github.com/lunarway/release-manager/internal/http"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/weaveworks/flux/event"
	fluxevent "github.com/weaveworks/flux/event"
	"github.com/weaveworks/flux/update"
)

type Message struct {
	Event event.Event
}

// Exporter sends a formatted event to an upstream.
type Exporter interface {
	// Send a message through the exporter.
	Send(c context.Context, event fluxevent.Event) error
}
type ReleaseManagerExporter struct {
	Log         *log.Logger
	Url         string
	AuthToken   string
	Environment string
}

func (f *ReleaseManagerExporter) Send(_ context.Context, event fluxevent.Event) error {
	f.Log.With("FluxEvent", event).Infof("flux event logged")
	client := &http.Client{
		Timeout: 20 * time.Second,
	}
	b := &bytes.Buffer{}
	err := json.NewEncoder(b).Encode(httpinternal.FluxNotifyRequest{
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
	})
	if err != nil {
		log.Errorf("error encoding FluxNotifyRequest: %+v", err)
		return err
	}
	url := f.Url + "/webhook/daemon/flux"
	req, err := http.NewRequest(http.MethodPost, url, b)
	if err != nil {
		log.Errorf("error generating FluxNotifyRequest to %s: %+v", url, err)
		return err
	}
	req.Header.Set("Authorization", "Bearer "+f.AuthToken)
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("error posting FluxNotifyRequest to %s: %+v", url, err)
		return err
	}
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("failed to read response body: %+v", err)
		}
		log.Errorf("release-manager returned %s status-code in flux ReleaseManagerExporter notify webhook: %s", resp.Status, body)
		return err
	}

	return nil
}

func getCommits(meta fluxevent.EventMetadata) []fluxevent.Commit {
	switch v := meta.(type) {
	case *fluxevent.CommitEventMetadata:
		return []fluxevent.Commit{
			fluxevent.Commit{
				Revision: v.Revision,
			},
		}
	case *fluxevent.SyncEventMetadata:
		return v.Commits
	default:
		return []fluxevent.Commit{}
	}
}

func getResult(meta fluxevent.EventMetadata) update.Result {
	switch v := meta.(type) {
	case *fluxevent.AutoReleaseEventMetadata:
		return v.Result
	case *fluxevent.ReleaseEventMetadata:
		return v.Result
	default:
		return update.Result{}
	}
}

func getChangedImages(meta fluxevent.EventMetadata) []string {
	switch v := meta.(type) {
	case *fluxevent.AutoReleaseEventMetadata:
		return v.Result.ChangedImages()
	case *fluxevent.ReleaseEventMetadata:
		return v.Result.ChangedImages()
	default:
		return []string{}
	}
}

func getErrors(meta fluxevent.EventMetadata) []fluxevent.ResourceError {
	switch v := meta.(type) {
	case *fluxevent.SyncEventMetadata:
		return v.Errors
	default:
		return []fluxevent.ResourceError{}
	}
}
