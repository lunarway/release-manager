package http

const (
	IntentTypeRelease = "Release"
	IntentTypePromote = "Promote"
)

type Intent struct {
	IntentType string        `json:"intentType,omitempty"`
	Release    ReleaseIntent `json:"release,omitempty"`
	Promote    PromoteIntent `json:"promote,omitempty"`
}

type ReleaseIntent struct {
	Branch     string `json:"branch,omitempty"`
	ArtifactID string `json:"artifactId,omitempty"`
}

type PromoteIntent struct {
	FromEnvironment string `json:"fromEnvironment,omitempty"`
}

func NewReleaseArtifactIntent(artifactID string) Intent {
	return Intent{
		IntentType: IntentTypeRelease,
		Release: ReleaseIntent{
			ArtifactID: artifactID,
		},
	}
}

func NewReleaseBranchIntent(branch string) Intent {
	return Intent{
		IntentType: IntentTypeRelease,
		Release: ReleaseIntent{
			Branch: branch,
		},
	}
}

func NewPromoteEnvironmentIntent(fromEnvironment string) Intent {
	return Intent{
		IntentType: IntentTypeRelease,
		Promote: PromoteIntent{
			FromEnvironment: fromEnvironment,
		},
	}
}
