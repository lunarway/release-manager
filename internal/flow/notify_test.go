package flow

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCommitMessageExtraction(t *testing.T) {
	tt := []struct {
		name          string
		commitMessage string
		expected      FluxReleaseMessage
		err           error
	}{
		{
			name:          "only four values match",
			commitMessage: "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			expected:      FluxReleaseMessage{},
			err:           errors.New("not enough matches"),
		},
		{
			name:          "exact values without newlines",
			commitMessage: "[env/service-name] release master-a037e03657-efc17d9df7 by author@lunar.app Artifact-created-by: Author <author@lunar.app> Artifact-released-by: Committer <committer@lunar.app>",
			expected: FluxReleaseMessage{
				Environment:  "env",
				Service:      "service-name",
				ArtifactID:   "master-a037e03657-efc17d9df7",
				GitAuthor:    "author@lunar.app",
				GitCommitter: "committer@lunar.app",
			},
			err: nil,
		},
		{
			name: "exact values with newlines",
			commitMessage: `[env/service-name] release master-a037e03657-efc17d9df7 by author@lunar.app
Artifact-created-by: Author <author@lunar.app>
Artifact-released-by: Committer <committer@lunar.app>`,
			expected: FluxReleaseMessage{
				Environment:  "env",
				Service:      "service-name",
				ArtifactID:   "master-a037e03657-efc17d9df7",
				GitAuthor:    "author@lunar.app",
				GitCommitter: "committer@lunar.app",
			},
			err: nil,
		},
		{
			name:          "only three values match",
			commitMessage: "[env/service-name] release master-1234567890-1234567890",
			expected:      FluxReleaseMessage{},
			err:           errors.New("not enough matches"),
		},
		{
			name:          "random commit message",
			commitMessage: "test test test test",
			expected:      FluxReleaseMessage{},
			err:           errors.New("not enough matches"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			output, err := parseCommitMessage(tc.commitMessage)
			if tc.err != nil {
				assert.EqualError(t, errors.Cause(err), tc.err.Error(), "output error not as expected")
			} else {
				assert.NoError(t, err, "no output error expected")
			}
			assert.Equal(t, tc.expected, output, "output logs not as expected")
		})
	}
}
