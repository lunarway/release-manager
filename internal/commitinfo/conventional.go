package commitinfo

import (
	"fmt"
	"strings"

	"github.com/lunarway/release-manager/internal/regexp"
	"github.com/pkg/errors"
)

type ConventionalCommitInfo struct {
	Message     string
	Description string
	Fields      []Field
}

type Field struct {
	Name  string
	Value string
}

func NewField(name, value string) Field {
	return Field{Name: name, Value: value}
}

func (i ConventionalCommitInfo) Field(name string) string {
	for _, field := range i.Fields {
		if field.Name == name {
			return field.Value
		}
	}
	return ""
}
func (i *ConventionalCommitInfo) SetField(name string, value string) {
	for index, field := range i.Fields {
		if field.Name == name {
			i.Fields[index].Value = value
			return
		}
	}
	i.Fields = append(i.Fields, NewField(name, value))
}

func (i ConventionalCommitInfo) String() string {
	var txts []string
	if i.Message != "" {
		txts = append(txts, i.Message)
	}
	if i.Description != "" {
		if i.Message != "" {
			txts = append(txts, i.Description)
		}
	}
	var fieldLines []string
	for _, field := range i.Fields {
		fieldLines = append(fieldLines, fmt.Sprintf("%s: %s", field.Name, field.Value))
	}
	if len(fieldLines) > 0 {
		txts = append(txts, strings.Join(fieldLines, "\n"))
	}
	return strings.Join(txts, "\n\n")
}

func ParseConventionalCommit(commitMessage string) (ConventionalCommitInfo, error) {
	matches := conventionalCommitRegex.FindStringSubmatch(commitMessage)
	if matches == nil {
		return ConventionalCommitInfo{}, errors.Wrap(ErrNoMatch, "message does not match expected conventional commit structure")
	}

	message := strings.Trim(matches[conventionalCommitRegexLookup.Message], "\n")
	description := strings.Trim(matches[conventionalCommitRegexLookup.Description], "\n")
	fieldsText := strings.Trim(matches[conventionalCommitRegexLookup.Fields], "\n")

	var fields []Field
	if fieldsText != "" {
		fieldLines := strings.Split(fieldsText, "\n")
		for _, fieldLine := range fieldLines {
			fieldParts := strings.SplitN(fieldLine, ":", 2)
			if len(fieldParts) != 2 {
				return ConventionalCommitInfo{}, fmt.Errorf("field line '%s' in commit could not be parsed", fieldLine)
			}
			fields = append(fields, NewField(fieldParts[0], strings.Trim(fieldParts[1], " ")))
		}
	}

	return ConventionalCommitInfo{
		Message:     message,
		Description: description,
		Fields:      fields,
	}, nil
}

var conventionalCommitRegexLookup = struct {
	Message     int
	Description int
	Fields      int
}{}
var conventionalCommitRegex = regexp.MustCompile(`(?s)^((?P<Message>[^\n]*)(?P<Description>.*?)((?P<Fields>(\n[a-zA-Z\-]+:[^\n]*)(\n[a-zA-Z\-]+:[^\n]*)*))?)?\n*$`, &conventionalCommitRegexLookup)
