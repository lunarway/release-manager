package commitinfo

import (
	"github.com/lunarway/release-manager/internal/intent"
)

func ParseIntent(cci ConventionalCommitInfo) intent.Intent {
	switch cci.Field("Release-intent") {
	case intent.TypeReleaseBranch:
		return intent.NewReleaseBranch(cci.Field("Release-branch"))
	case intent.TypePromote:
		return intent.NewPromoteEnvironment(cci.Field("Release-environment"))
	case intent.TypeRollback:
		return intent.NewRollback(cci.Field("Release-rollback-of-artifact-id"))
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
		cci.SetField("Release-branch", intentObj.ReleaseBranch.Branch)
	case intent.TypePromote:
		cci.SetField("Release-environment", intentObj.Promote.FromEnvironment)
	case intent.TypeRollback:
		cci.SetField("Release-rollback-of-artifact-id", intentObj.Rollback.PreviousArtifactID)
	case intent.TypeAutoRelease:
		// nothing yet
	}
}
