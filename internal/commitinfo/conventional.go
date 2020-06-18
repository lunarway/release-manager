package commitinfo

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

type ConventionalCommitInfo struct {
	Message     string
	Description string
	Fields      map[string]string
}

func ParseCommit(commitMessage string) (ConventionalCommitInfo, error) {
	matches := conventionalCommitRegex.FindStringSubmatch(commitMessage)
	if matches == nil {
		return ConventionalCommitInfo{}, errors.New("no match")
	}

	message := strings.Trim(matches[conventionalCommitRegexNamesLookup["message"]], "\n")
	description := strings.Trim(matches[conventionalCommitRegexNamesLookup["description"]], "\n")
	fieldsText := strings.Trim(matches[conventionalCommitRegexNamesLookup["fields"]], "\n")

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

var conventionalCommitRegex = regexp.MustCompile(`(?s)^((?P<message>[^\n]+)(?P<description>.*?)((?P<fields>(\n[a-zA-Z\-]+:[^\n]+)(\n[a-zA-Z\-]+:[^\n]*)*))?)?\n*$`)
var conventionalCommitRegexNamesLookup = make(map[string]int)

func init() {
	for index, name := range conventionalCommitRegex.SubexpNames() {
		if name != "" {
			conventionalCommitRegexNamesLookup[name] = index
		}
	}
}
