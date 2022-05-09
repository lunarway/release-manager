package template

import (
	"encoding/json"
	"fmt"
	"io"
	"text/template"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/lunarway/release-manager/internal/intent"
)

// Output parses the template text as a Go template. The empty interface is
// available as the root object in the template.
//
// Some utility functions are available for data manipulation.
func Output(destination io.Writer, name, text string, data interface{}) error {
	t := template.New(name)
	t.Funcs(FuncMap())
	t, err := t.Parse(text)
	if err != nil {
		return fmt.Errorf("invalid template: %v", err)
	}
	return t.Execute(destination, data)
}

func tmplRightPad(s string, padding int) string {
	template := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(template, s)
}

func tmplMaxLength(list interface{}, key string) (int, error) {
	// this is a poor mans generic implementation of finding the maximum length of
	// a value based on a key. We change the type of the list to a
	// map[string]interface{} which makes key lookups safe.
	bytes, err := json.Marshal(list)
	if err != nil {
		return 0, err
	}

	var mapedList []map[string]interface{}
	err = json.Unmarshal(bytes, &mapedList)
	if err != nil {
		return 0, err
	}

	var max int
	for _, m := range mapedList {
		if s, ok := m[key].(string); ok {
			if len(s) > max {
				max = len(s)
			}
		}
	}

	return max, nil
}

func FuncMap() template.FuncMap {
	return template.FuncMap{
		"rightPad":  tmplRightPad,
		"maxLength": tmplMaxLength,
		"add": func(a, b int) int {
			return a + b
		},
		"dateFormat": func() string {
			return "2006-01-02 15:04:05"
		},
		"humanizeTime": func(input time.Time) string {
			return humanize.Time(input)
		},
	}
}

func IntentString(i intent.Intent) string {
	switch i.Type {
	case intent.TypeAutoRelease:
		return "autorelease"
	case intent.TypePromote:
		return fmt.Sprintf("promotion from %s", i.Promote.FromEnvironment)
	case intent.TypeReleaseArtifact:
		return "artifact release"
	case intent.TypeReleaseBranch:
		return fmt.Sprintf("%s branch release", i.ReleaseBranch.Branch)
	case intent.TypeRollback:
		return fmt.Sprintf("rollback of %s", i.Rollback.PreviousArtifactID)
	default:
		return fmt.Sprintf("unknown intent type '%s'", i.Type)
	}
}
