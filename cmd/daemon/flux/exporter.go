package flux

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	httpinternal "github.com/lunarway/release-manager/internal/http"
	"github.com/pkg/errors"

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
	Url         string
	AuthToken   string
	Environment string
}

func (f *ReleaseManagerExporter) Send(_ context.Context, event event.Event) error {
	f.Log.With("event", event).Infof("flux event logged")
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
		return errors.WithMessage(err, "encoding FluxNotifyRequest")
	}
	url := f.Url + "/webhook/daemon/flux"
	req, err := http.NewRequest(http.MethodPost, url, b)
	if err != nil {
		return errors.WithMessage(err, "error generating FluxNotifyRequest")
	}
	req.Header.Set("Authorization", "Bearer "+f.AuthToken)
	resp, err := client.Do(req)
	if err != nil {
		return errors.WithMessage(err, "error posting FluxNotifyRequest")
	}
	if resp.StatusCode != 200 {
		_, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("failed to read response body: %+v", err)
		}
		return errors.WithMessage(err, fmt.Sprintf("release-manager returned %s status-code in flux ReleaseManagerExporter notify webhook", resp.Status))
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
