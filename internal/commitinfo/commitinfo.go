package commitinfo

import (
	"github.com/lunarway/release-manager/internal/regexp"
	"github.com/pkg/errors"
)

type CommitInfo struct {
	ArtifactID  string
	AuthorName  string
	AuthorEmail string
	Service     string
}

func ExtractInfoFromCommit(commitMessage string) (CommitInfo, error) {
	convInfo, err := ParseCommit(commitMessage)
	if err != nil {
		return CommitInfo{}, err
	}

	matches := extractInfoFromCommitMessageRegex.FindStringSubmatch(convInfo.Message)
	if matches == nil {
		return CommitInfo{}, ErrNoMatch
	}
	author, err := ParsePerson(convInfo.Fields["Artifact-created-by"])
	if err != nil && !errors.Is(err, ErrNoMatch) {
		return CommitInfo{}, err
	}

	if matches[extractInfoFromCommitMessageRegexLookup.Type] != "artifact" {
		return CommitInfo{}, ErrNoMatch
	}

	return CommitInfo{
		//Type:        matches[extractInfoFromCommitMessageRegexLookup.Type],
		Service:     matches[extractInfoFromCommitMessageRegexLookup.Service],
		ArtifactID:  matches[extractInfoFromCommitMessageRegexLookup.ArtifactID],
		AuthorName:  author.Name,
		AuthorEmail: author.Email,
	}, nil
}

var extractInfoFromCommitMessageRegexLookup = struct {
	Service    int
	ArtifactID int
	Type       int
}{}
var extractInfoFromCommitMessageRegex = regexp.MustCompile(`^\[(?P<Service>.*)\] (?P<Type>[a-z]+) (?P<ArtifactID>[^ ]+) by .*$`, &extractInfoFromCommitMessageRegexLookup)
