package intent

const (
	TypeRelease     = "Release"
	TypePromote     = "Promote"
	TypeRollback    = "Rollback"
	TypeAutoRelease = "AutoRelease"
)

type Intent struct {
	Type    string        `json:"intentType,omitempty"`
	Release ReleaseIntent `json:"release,omitempty"`
	Promote PromoteIntent `json:"promote,omitempty"`
}

type ReleaseIntent struct {
	Branch     string `json:"branch,omitempty"`
	ArtifactID string `json:"artifactId,omitempty"`
}

type PromoteIntent struct {
	FromEnvironment string `json:"fromEnvironment,omitempty"`
}

func NewReleaseArtifact(artifactID string) Intent {
	return Intent{
		Type: TypeRelease,
		Release: ReleaseIntent{
			ArtifactID: artifactID,
		},
	}
}

func NewReleaseBranch(branch string) Intent {
	return Intent{
		Type: TypeRelease,
		Release: ReleaseIntent{
			Branch: branch,
		},
	}
}

func NewPromoteEnvironment(fromEnvironment string) Intent {
	return Intent{
		Type: TypeRelease,
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

func NewRollback() Intent {
	return Intent{
		Type: TypeRollback,
	}
}

func (intent *Intent) Valid() bool {
	return !intent.Empty()
}

func (intent *Intent) Empty() bool {
	return intent.Type == ""
}
