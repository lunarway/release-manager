package commitinfo

import (
	"regexp"

	"github.com/pkg/errors"
)

type CommitInfo struct {
	ArtifactID  string
	AuthorName  string
	AuthorEmail string
	Service     string
}

func ExtractInfoFromCommit() func(string) (CommitInfo, error) {
	extractInfoFromCommitRegex := regexp.MustCompile(`^\[(?P<service>.*)\] artifact (?P<artifactID>[^ ]+) by .*\nArtifact-created-by:\s(?P<authorName>.*)\s<(?P<authorEmail>.*)>`)
	extractInfoFromCommitRegexNamesLookup := make(map[string]int)
	for index, name := range extractInfoFromCommitRegex.SubexpNames() {
		if name != "" {
			extractInfoFromCommitRegexNamesLookup[name] = index
		}
	}

	return func(message string) (CommitInfo, error) {
		matches := extractInfoFromCommitRegex.FindStringSubmatch(message)
		if matches == nil {
			return CommitInfo{}, errors.New("no match")
		}
		return CommitInfo{
			Service:     matches[extractInfoFromCommitRegexNamesLookup["service"]],
			ArtifactID:  matches[extractInfoFromCommitRegexNamesLookup["artifactID"]],
			AuthorName:  matches[extractInfoFromCommitRegexNamesLookup["authorName"]],
			AuthorEmail: matches[extractInfoFromCommitRegexNamesLookup["authorEmail"]],
		}, nil
	}
}
