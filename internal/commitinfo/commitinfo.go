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

func ExtractInfoFromCommit(commitMessage string) (CommitInfo, error) {
	matches := extractInfoFromCommitRegex.FindStringSubmatch(commitMessage)
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

var extractInfoFromCommitRegex = regexp.MustCompile(`^\[(?P<service>.*)\] artifact (?P<artifactID>[^ ]+) by .*\nArtifact-created-by:\s(?P<authorName>.*)\s<(?P<authorEmail>.*)>`)
var extractInfoFromCommitRegexNamesLookup = make(map[string]int)

func init() {
	for index, name := range extractInfoFromCommitRegex.SubexpNames() {
		if name != "" {
			extractInfoFromCommitRegexNamesLookup[name] = index
		}
	}
}
