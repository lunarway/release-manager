package command

import (
	"fmt"
	"io"
	"text/template"
)

// templateOutput parses the template text as a Go template. The empty interface
// is available as the root object in the template.
//
// Some utility functions are available for data manipulation.
func templateOutput(destination io.Writer, name, text string, data interface{}) error {
	t := template.New(name)
	t.Funcs(template.FuncMap{
		"rightPad": tmplRightPad,
	})
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
