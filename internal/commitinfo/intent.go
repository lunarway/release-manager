package commitinfo

import (
	"github.com/lunarway/release-manager/internal/intent"
)

func ParseIntent(cci ConventionalCommitInfo) intent.Intent {
	switch cci.Fields["Release-intent"] {
	case intent.TypeReleaseBranch:
		return intent.NewReleaseBranch(cci.Fields["Release-branch"])
	case intent.TypePromote:
		return intent.NewPromoteEnvironment(cci.Fields["Release-environment"])
	case intent.TypeRollback:
		return intent.NewRollback(cci.Fields["Release-rollback-of-artifact-id"])
	case intent.TypeAutoRelease:
		return intent.NewAutoRelease()
	default:
		return intent.NewReleaseArtifact()
	}
}

func AddIntentToConventionalCommitInfo(intentObj intent.Intent, cci *ConventionalCommitInfo) {
	cci.Fields["Release-intent"] = intentObj.Type
	switch intentObj.Type {
	case intent.TypeReleaseBranch:
		cci.Fields["Release-branch"] = intentObj.ReleaseBranch.Branch
	case intent.TypePromote:
		cci.Fields["Release-environment"] = intentObj.Promote.FromEnvironment
	case intent.TypeRollback:
		cci.Fields["Release-rollback-of-artifact-id"] = intentObj.Rollback.PreviousArtifactID
	case intent.TypeAutoRelease:
		// nothing yet
	}
}
