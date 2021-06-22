package commitinfo

import (
	"github.com/lunarway/release-manager/internal/intent"
)

const (
	FieldReleaseIntent           = "Release-intent"
	FieldRollbackOfArtifactId    = "Rollback-of-artifact-id"
	FieldReleaseOfBranch         = "Release-of-branch"
	FieldPromotedFromEnvironment = "Promoted-from-environment"
)

func parseIntent(cci ConventionalCommitInfo, commitMessageMatches []string) intent.Intent {
	switch cci.Field(FieldReleaseIntent) {
	case intent.TypeReleaseBranch:
		return intent.NewReleaseBranch(cci.Field(FieldReleaseOfBranch))
	case intent.TypePromote:
		return intent.NewPromoteEnvironment(cci.Field(FieldPromotedFromEnvironment))
	case intent.TypeRollback:
		return intent.NewRollback(cci.Field(FieldRollbackOfArtifactId))
	case intent.TypeAutoRelease:
		return intent.NewAutoRelease()
	default:
		// A check for compatability reasons, for back when only the message was saying it was a rollback.
		if commitMessageMatches != nil && commitMessageMatches[parseCommitInfoFromCommitMessageRegexLookup.Type] == "rollback" {
			return intent.NewRollback(commitMessageMatches[parseCommitInfoFromCommitMessageRegexLookup.PreviousArtifactID])
		}
		return intent.NewReleaseArtifact()
	}
}

func addIntentToConventionalCommitInfo(intentObj intent.Intent, cci *ConventionalCommitInfo) {
	cci.SetField(FieldReleaseIntent, intentObj.Type)
	switch intentObj.Type {
	case intent.TypeReleaseBranch:
		cci.SetField(FieldReleaseOfBranch, intentObj.ReleaseBranch.Branch)
	case intent.TypePromote:
		cci.SetField(FieldPromotedFromEnvironment, intentObj.Promote.FromEnvironment)
	case intent.TypeRollback:
		cci.SetField(FieldRollbackOfArtifactId, intentObj.Rollback.PreviousArtifactID)
	case intent.TypeAutoRelease:
		// nothing yet
	}
}
