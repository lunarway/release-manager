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
	Fields      map[string]string
}

func (i ConventionalCommitInfo) String() string {
	txt := i.Message
	if txt != "" && i.Description != "" {
		txt = fmt.Sprintf("%s\n\n%s", txt, i.Description)
	}
	var fieldLines []string
	for field, value := range i.Fields {
		fieldLines = append(fieldLines, fmt.Sprintf("%s: %s", field, value))
	}
	if txt != "" && len(fieldLines) > 0 {
		txt = fmt.Sprintf("%s\n\n%s", txt, strings.Join(fieldLines, "\n"))
	}
	return txt
}

func ParseConventionalCommit(commitMessage string) (ConventionalCommitInfo, error) {
	matches := conventionalCommitRegex.FindStringSubmatch(commitMessage)
	if matches == nil {
		return ConventionalCommitInfo{}, errors.Wrap(ErrNoMatch, "message does not match expected conventional commit structure")
	}

	message := strings.Trim(matches[conventionalCommitRegexLookup.Message], "\n")
	description := strings.Trim(matches[conventionalCommitRegexLookup.Description], "\n")
	fieldsText := strings.Trim(matches[conventionalCommitRegexLookup.Fields], "\n")

	fields := make(map[string]string)
	if fieldsText != "" {
		fieldLines := strings.Split(fieldsText, "\n")
		for _, fieldLine := range fieldLines {
			fieldParts := strings.SplitN(fieldLine, ": ", 2)
			if len(fieldParts) != 2 {
				return ConventionalCommitInfo{}, fmt.Errorf("field line '%s' in commmit could not be parsed", fieldLine)
			}
			fields[fieldParts[0]] = fieldParts[1]
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
var conventionalCommitRegex = regexp.MustCompile(`(?s)^((?P<Message>[^\n]+)(?P<Description>.*?)((?P<Fields>(\n[a-zA-Z\-]+:[^\n]+)(\n[a-zA-Z\-]+:[^\n]*)*))?)?\n*$`, &conventionalCommitRegexLookup)
