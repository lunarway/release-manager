package flux

import (
	"encoding/json"
	"errors"
	"fmt"
)

/*
	This file contains models copied from github.com/fluxcd/flux for deserializing
	events from flux websocket connections. It is maintained here to limit the
	import scope to only the fields we use and this limit transitive
	dependenencies of the flux project.
*/

const (
	EventCommit = "commit"
	EventSync   = "sync"
)

type Event struct {
	// Type is the type of event, usually "release" for now, but could be other
	// things later
	Type string `json:"type"`

	// Metadata is Event.Type-specific metadata. If an event has no metadata,
	// this will be nil.
	Metadata EventMetadata `json:"metadata,omitempty"`
}

func (e Event) String() string {
	switch e.Type {
	case EventCommit:
		metadata := e.Metadata.(*CommitEventMetadata)
		svcStr := "<no changes>"
		return fmt.Sprintf("Commit: %s, %s", shortRevision(metadata.Revision), svcStr)
	case EventSync:
		metadata := e.Metadata.(*SyncEventMetadata)
		revStr := "<no revision>"
		if 0 < len(metadata.Commits) && len(metadata.Commits) <= 2 {
			revStr = shortRevision(metadata.Commits[0].Revision)
		} else if len(metadata.Commits) > 2 {
			revStr = fmt.Sprintf(
				"%s..%s",
				shortRevision(metadata.Commits[len(metadata.Commits)-1].Revision),
				shortRevision(metadata.Commits[0].Revision),
			)
		}
		svcStr := "no workloads changed"
		return fmt.Sprintf("Sync: %s, %s", revStr, svcStr)
	default:
		return fmt.Sprintf("Unknown event: %s", e.Type)
	}
}

func shortRevision(rev string) string {
	if len(rev) <= 7 {
		return rev
	}
	return rev[:7]
}

// CommitEventMetadata is the metadata for when new git commits are created
type CommitEventMetadata struct {
	Revision string `json:"revision,omitempty"`
}

// Commit represents the commit information in a sync event. We could
// use git.Commit, but that would lead to an import cycle, and may
// anyway represent coupling (of an internal API to serialised data)
// that we don't want.
type Commit struct {
	Revision string `json:"revision"`
	Message  string `json:"message"`
}

type ResourceError struct {
	// ID    ID
	Path  string
	Error string
}

// SyncEventMetadata is the metadata for when new a commit is synced to the cluster
type SyncEventMetadata struct {
	Commits []Commit        `json:"commits,omitempty"`
	Errors  []ResourceError `json:"errors,omitempty"`
}

type UnknownEventMetadata map[string]interface{}

func (e *Event) UnmarshalJSON(in []byte) error {
	type alias Event
	var wireEvent struct {
		*alias
		MetadataBytes json.RawMessage `json:"metadata,omitempty"`
	}
	wireEvent.alias = (*alias)(e)

	// Now unmarshall custom wireEvent with RawMessage
	if err := json.Unmarshal(in, &wireEvent); err != nil {
		return err
	}
	if wireEvent.Type == "" {
		return errors.New("Event type is empty")
	}

	// The cases correspond to kinds of event that we care about
	// processing e.g., for notifications.
	switch wireEvent.Type {

	case EventCommit:
		var metadata CommitEventMetadata
		if err := json.Unmarshal(wireEvent.MetadataBytes, &metadata); err != nil {
			return err
		}
		e.Metadata = &metadata
		break
	case EventSync:
		var metadata SyncEventMetadata
		if err := json.Unmarshal(wireEvent.MetadataBytes, &metadata); err != nil {
			return err
		}
		e.Metadata = &metadata
		break
	default:
		if len(wireEvent.MetadataBytes) > 0 {
			var metadata UnknownEventMetadata
			if err := json.Unmarshal(wireEvent.MetadataBytes, &metadata); err != nil {
				return err
			}
			e.Metadata = metadata
		}
	}

	// By default, leave the Event Metadata as map[string]interface{}
	return nil
}

// EventMetadata is a type safety trick used to make sure that Metadata field
// of Event is always a pointer, so that consumers can cast without being
// concerned about encountering a value type instead. It works by virtue of the
// fact that the method is only defined for pointer receivers; the actual
// method chosen is entirely arbitrary.
type EventMetadata interface {
	Type() string
}

func (cem *CommitEventMetadata) Type() string {
	return EventCommit
}

func (cem *SyncEventMetadata) Type() string {
	return EventSync
}

// Special exception from pointer receiver rule, as UnknownEventMetadata is a
// type alias for a map
func (uem UnknownEventMetadata) Type() string {
	return "unknown"
}
