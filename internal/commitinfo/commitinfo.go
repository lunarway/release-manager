package commitinfo

import (
	"fmt"

	"github.com/lunarway/release-manager/internal/intent"
	"github.com/lunarway/release-manager/internal/regexp"
	"github.com/pkg/errors"
)

type CommitInfo struct {
	ArtifactID        string
	ArtifactCreatedBy PersonInfo
	ReleasedBy        PersonInfo
	Service           string
	Environment       string
	Intent            intent.Intent
}

func (i CommitInfo) String() string {
	cci := ConventionalCommitInfo{
		Message: fmt.Sprintf("[%s/%s] %s %s by %s", i.Environment, i.Service, "release", i.ArtifactID, i.ReleasedBy.Email),
		Fields: map[string]string{
			"Artifact-released-by": i.ReleasedBy.String(),
			"Artifact-created-by":  i.ArtifactCreatedBy.String(),
		},
	}

	AddIntentToConventionalCommitInfo(i.Intent, &cci)

	return cci.String()
}

func ParseCommitInfo(commitMessage string) (CommitInfo, error) {
	convInfo, err := ParseConventionalCommit(commitMessage)
	if err != nil {
		return CommitInfo{}, err
	}

	matches := extractInfoFromCommitMessageRegex.FindStringSubmatch(convInfo.Message)
	if matches == nil {
		return CommitInfo{}, errors.Wrap(ErrNoMatch, fmt.Sprintf("commit message '%s' does not match expected message structure", convInfo.Message))
	}
	artifactCreatedBy, err := ParsePerson(convInfo.Fields["Artifact-created-by"])
	if err != nil && !errors.Is(err, ErrNoMatch) {
		return CommitInfo{}, errors.Wrap(err, fmt.Sprintf("commit got unknown parsing error of %s with content '%s'", "Artifact-created-by", convInfo.Fields["Artifact-created-by"]))
	}
	releasedBy, err := ParsePerson(convInfo.Fields["Artifact-released-by"])
	if err != nil && !errors.Is(err, ErrNoMatch) {
		return CommitInfo{}, errors.Wrap(err, fmt.Sprintf("commit got unknown parsing error of %s with content '%s'", "Artifact-released-by", convInfo.Fields["Artifact-released-by"]))
	}

	intentObj := ParseIntent(convInfo)
	if matches[extractInfoFromCommitMessageRegexLookup.Type] == "artifact" {
		return CommitInfo{}, errors.Wrap(ErrNoMatch, fmt.Sprintf("commit type '%s' is not considered a match", matches[extractInfoFromCommitMessageRegexLookup.Type]))
	}

	return CommitInfo{
		Intent:            intentObj,
		Environment:       matches[extractInfoFromCommitMessageRegexLookup.Environment],
		Service:           matches[extractInfoFromCommitMessageRegexLookup.Service],
		ArtifactID:        matches[extractInfoFromCommitMessageRegexLookup.ArtifactID],
		ArtifactCreatedBy: artifactCreatedBy,
		ReleasedBy:        releasedBy,
	}, nil
}

var extractInfoFromCommitMessageRegexLookup = struct {
	Environment int
	Service     int
	ArtifactID  int
	Type        int
}{}
var extractInfoFromCommitMessageRegex = regexp.MustCompile(`^\[(?P<Environment>[^/]+)/(?P<Service>.*)\] (?P<Type>[a-z]+) (?P<ArtifactID>[^ ]+) by .*$`, &extractInfoFromCommitMessageRegexLookup)
