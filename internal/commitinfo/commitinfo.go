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
	artifactCreatedBy, err := ParsePerson(convInfo.Field("Artifact-created-by"))
	if err != nil && !errors.Is(err, ErrNoMatch) {
		return CommitInfo{}, errors.Wrap(err, fmt.Sprintf("commit got unknown parsing error of %s with content '%s'", "Artifact-created-by", convInfo.Field("Artifact-created-by")))
	}
	releasedBy, err := ParsePerson(convInfo.Field("Artifact-released-by"))
	if err != nil && !errors.Is(err, ErrNoMatch) {
		return CommitInfo{}, errors.Wrap(err, fmt.Sprintf("commit got unknown parsing error of %s with content '%s'", "Artifact-released-by", convInfo.Field("Artifact-released-by")))
	}

	intentObj := ParseIntent(convInfo)
	if matches[extractInfoFromCommitMessageRegexLookup.Type] == "artifact" {
		return CommitInfo{}, errors.Wrap(ErrNoMatch, fmt.Sprintf("commit type '%s' is not considered a match", matches[extractInfoFromCommitMessageRegexLookup.Type]))
	}

	service := convInfo.Field(FieldService)
	if service == "" {
		service = matches[extractInfoFromCommitMessageRegexLookup.Service]
	}
	environment := convInfo.Field(FieldEnvironment)
	if environment == "" {
		environment = matches[extractInfoFromCommitMessageRegexLookup.Environment]
	}
	artifactID := convInfo.Field(FieldArtifactID)
	if artifactID == "" {
		artifactID = matches[extractInfoFromCommitMessageRegexLookup.ArtifactID]
	}
	if releasedBy.Email == "" {
		releasedBy = NewPersonInfo("", matches[extractInfoFromCommitMessageRegexLookup.ReleaseByEmail])
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

var extractInfoFromCommitMessageRegexLookup = struct {
	Environment    int
	Service        int
	ArtifactID     int
	Type           int
	ReleaseByEmail int
}{}
var extractInfoFromCommitMessageRegex = regexp.MustCompile(`^\[(?P<Environment>[^/]+)/(?P<Service>.*)\] (?P<Type>[a-z]+) (?P<ArtifactID>[^ ]+) by (?P<ReleaseByEmail>.*)$`, &extractInfoFromCommitMessageRegexLookup)
