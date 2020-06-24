package commitinfo

import (
	"github.com/lunarway/release-manager/internal/intent"
)

const (
	FieldRollbackOfArtifactId    = "Rollback-of-artifact-id"
	FieldReleaseOfBranch         = "Release-of-branch"
	FieldPromotedFromEnvironment = "Promoted-from-environment"
)

func ParseIntent(cci ConventionalCommitInfo) intent.Intent {
	switch cci.Field("Release-intent") {
	case intent.TypeReleaseBranch:
		return intent.NewReleaseBranch(cci.Field(FieldReleaseOfBranch))
	case intent.TypePromote:
		return intent.NewPromoteEnvironment(cci.Field(FieldPromotedFromEnvironment))
	case intent.TypeRollback:
		return intent.NewRollback(cci.Field(FieldRollbackOfArtifactId))
	case intent.TypeAutoRelease:
		return intent.NewAutoRelease()
	default:
		return intent.NewReleaseArtifact()
	}
}

func AddIntentToConventionalCommitInfo(intentObj intent.Intent, cci *ConventionalCommitInfo) {
	cci.SetField("Release-intent", intentObj.Type)
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
