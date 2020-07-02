package commitinfo

import (
	"fmt"

	"github.com/lunarway/release-manager/internal/intent"
	"github.com/lunarway/release-manager/internal/regexp"
	"github.com/pkg/errors"
)

const (
	FieldService            = "Service"
	FieldEnvironment        = "Environment"
	FieldArtifactID         = "Artifact-ID"
	FieldArtifactReleasedBy = "Artifact-released-by"
	FieldArtifactCreatedBy  = "Artifact-created-by"
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
	var releaseType string
	switch i.Intent.Type {
	case intent.TypeRollback:
		releaseType = "rollback"
	case intent.TypeAutoRelease:
		releaseType = "auto release"
	default:
		releaseType = "release"
	}

	cci := ConventionalCommitInfo{
		Message: fmt.Sprintf("[%s/%s] %s %s by %s", i.Environment, i.Service, releaseType, i.ArtifactID, i.ReleasedBy.Email),
		Fields: []Field{
			NewField(FieldService, i.Service),
			NewField(FieldEnvironment, i.Environment),
			NewField(FieldArtifactID, i.ArtifactID),
			NewField(FieldArtifactReleasedBy, i.ReleasedBy.String()),
			NewField(FieldArtifactCreatedBy, i.ArtifactCreatedBy.String()),
		},
	}

	addIntentToConventionalCommitInfo(i.Intent, &cci)

	return cci.String()
}

// ParseCommitInfo takes a full git commit message in the ConventionalCommit format and tries to extract release information from it
// It will extract it using conventional commit fields first, but if the correct fields isn't there it will fallback to parsing message
// for backward compatability reasons.
// The following backward-compatibility actions are done in the parsing (if not found in the fields!):
// * If `artifact` is written in title, the commit info is considered a "no match"
// * If `rollback` is written in title, the Intent is considered to be rollback intent, as well as the PreviousArtifactID is attempted
//   to be extracted
// * Environment and Service name is extracted from the `[<env>/<service>]`-brackets in the title
// * User email in title is interpreted as releaser
func ParseCommitInfo(commitMessage string) (CommitInfo, error) {
	convInfo, err := ParseConventionalCommit(commitMessage)
	if err != nil {
		return CommitInfo{}, err
	}

	matches := parseCommitInfoFromCommitMessageRegex.FindStringSubmatch(convInfo.Message)
	if matches == nil && !convInfo.HasField(FieldReleaseIntent) {
		return CommitInfo{}, errors.Wrap(ErrNoMatch, fmt.Sprintf("commit message '%s' do not have a Release-intent field and did not match expected message structure", convInfo.Message))
	}

	if matches != nil && matches[parseCommitInfoFromCommitMessageRegexLookup.Type] == "artifact" {
		return CommitInfo{}, errors.Wrap(ErrNoMatch, fmt.Sprintf("commit type '%s' is not considered a match", matches[parseCommitInfoFromCommitMessageRegexLookup.Type]))
	}

	artifactCreatedBy, err := ParsePerson(convInfo.Field("Artifact-created-by"))
	if err != nil && !errors.Is(err, ErrNoMatch) {
		return CommitInfo{}, errors.Wrap(err, fmt.Sprintf("commit got unknown parsing error of %s with content '%s'", "Artifact-created-by", convInfo.Field("Artifact-created-by")))
	}
	releasedBy, err := ParsePerson(convInfo.Field("Artifact-released-by"))
	if err != nil && !errors.Is(err, ErrNoMatch) {
		return CommitInfo{}, errors.Wrap(err, fmt.Sprintf("commit got unknown parsing error of %s with content '%s'", "Artifact-released-by", convInfo.Field("Artifact-released-by")))
	}
	intentObj := parseIntent(convInfo, matches)

	service := convInfo.Field(FieldService)
	if matches != nil && service == "" {
		service = matches[parseCommitInfoFromCommitMessageRegexLookup.Service]
	}
	environment := convInfo.Field(FieldEnvironment)
	if matches != nil && environment == "" {
		environment = matches[parseCommitInfoFromCommitMessageRegexLookup.Environment]
	}
	artifactID := convInfo.Field(FieldArtifactID)
	if matches != nil && artifactID == "" {
		artifactID = matches[parseCommitInfoFromCommitMessageRegexLookup.ArtifactID]
	}
	if matches != nil && releasedBy.Email == "" {
		releasedBy = NewPersonInfo("", matches[parseCommitInfoFromCommitMessageRegexLookup.ReleaseByEmail])
	}

	return CommitInfo{
		Intent:            intentObj,
		Service:           service,
		Environment:       environment,
		ArtifactID:        artifactID,
		ArtifactCreatedBy: artifactCreatedBy,
		ReleasedBy:        releasedBy,
	}, nil
}

var parseCommitInfoFromCommitMessageRegexLookup = struct {
	Environment        int
	Service            int
	ArtifactID         int
	PreviousArtifactID int
	Type               int
	ReleaseByEmail     int
}{}
var parseCommitInfoFromCommitMessageRegex = regexp.MustCompile(`^\[(?P<Environment>[^/]+)/(?P<Service>.*)\] (?P<Type>[a-z]+)( (?P<PreviousArtifactID>[^ ]+) to)? (?P<ArtifactID>[^ ]+)( by (?P<ReleaseByEmail>.*))?$`, &parseCommitInfoFromCommitMessageRegexLookup)
