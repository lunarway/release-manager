package flux

import "github.com/weaveworks/flux/event"

type Message struct {
	TitleLink string
	Body      string
	Type      string
	Title     string
	Event     event.Event
}
