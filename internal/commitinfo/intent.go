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
		return intent.NewRollback()
	case intent.TypeAutoRelease:
		return intent.NewAutoRelease()
	default:
		return intent.NewReleaseArtifact()
	}
}
