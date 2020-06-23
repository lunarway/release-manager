package intent

import "fmt"

const (
	TypeReleaseArtifact = "ReleaseArtifact"
	TypeReleaseBranch   = "ReleaseBranch"
	TypePromote         = "Promote"
	TypeRollback        = "Rollback"
	TypeAutoRelease     = "AutoRelease"
)

type Intent struct {
	Type          string              `json:"type,omitempty"`
	ReleaseBranch ReleaseBranchIntent `json:"releaseBranch,omitempty"`
	Promote       PromoteIntent       `json:"promote,omitempty"`
	Rollback      RollbackIntent      `json:"rollback,omitempty"`
}

type ReleaseBranchIntent struct {
	Branch string `json:"branch,omitempty"`
}

type PromoteIntent struct {
	FromEnvironment string `json:"fromEnvironment,omitempty"`
}

type RollbackIntent struct {
	PreviousArtifactID string `json:"previousArtifactId,omitempty"`
}

func NewReleaseArtifact() Intent {
	return Intent{
		Type: TypeReleaseArtifact,
	}
}

func NewReleaseBranch(branch string) Intent {
	return Intent{
		Type: TypeReleaseBranch,
		ReleaseBranch: ReleaseBranchIntent{
			Branch: branch,
		},
	}
}

func NewPromoteEnvironment(fromEnvironment string) Intent {
	return Intent{
		Type: TypePromote,
		Promote: PromoteIntent{
			FromEnvironment: fromEnvironment,
		},
	}
}

func NewAutoRelease() Intent {
	return Intent{
		Type: TypeAutoRelease,
	}
}

func NewRollback(previousArtifactID string) Intent {
	return Intent{
		Type: TypeRollback,
		Rollback: RollbackIntent{
			PreviousArtifactID: previousArtifactID,
		},
	}
}

func (intent *Intent) Valid() bool {
	return !intent.Empty()
}

func (intent *Intent) Empty() bool {
	return intent.Type == ""
}

func (intent *Intent) AsArtifactWithIntent(artifactID string) string {
	switch intent.Type {
	case TypeReleaseBranch:
		return fmt.Sprintf("branch '%s' with artifact '%s'", intent.ReleaseBranch.Branch, artifactID)
	case TypeReleaseArtifact:
		return fmt.Sprintf("artifact '%s'", artifactID)
	case TypePromote:
		return fmt.Sprintf("promotion from '%s' with artifact '%s'", intent.Promote.FromEnvironment, artifactID)
	case TypeRollback:
		return fmt.Sprintf("rollback to artifact '%s' from artifact '%s'", artifactID, intent.Rollback.PreviousArtifactID)
	case TypeAutoRelease:
		return fmt.Sprintf("autorelease artifact '%s'", artifactID)
	default:
		return fmt.Sprintf("invalid intent with artifact '%s'", artifactID)
	}
}
