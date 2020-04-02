package flux

import (
	"github.com/weaveworks/flux/event"
	"github.com/weaveworks/flux/update"
)

func GetCommits(meta event.EventMetadata) []event.Commit {
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

func GetResult(meta event.EventMetadata) update.Result {
	switch v := meta.(type) {
	case *event.AutoReleaseEventMetadata:
		return v.Result
	case *event.ReleaseEventMetadata:
		return v.Result
	default:
		return update.Result{}
	}
}

func GetChangedImages(meta event.EventMetadata) []string {
	switch v := meta.(type) {
	case *event.AutoReleaseEventMetadata:
		return v.Result.ChangedImages()
	case *event.ReleaseEventMetadata:
		return v.Result.ChangedImages()
	default:
		return []string{}
	}
}

func GetErrors(meta event.EventMetadata) []event.ResourceError {
	switch v := meta.(type) {
	case *event.SyncEventMetadata:
		return v.Errors
	default:
		return []event.ResourceError{}
	}
}
