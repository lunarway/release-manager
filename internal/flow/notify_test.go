package flow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommitMessageExtraction(t *testing.T) {
	tt := []struct {
		name          string
		commitMessage string
		expected      FluxReleaseMessage
	}{
		{
			name:          "exact values",
			commitMessage: "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			expected: FluxReleaseMessage{
				Environment: "env",
				Service:     "service-name",
				ArtifactID:  "master-1234567890-1234567890",
				GitAuthor:   "test@lunar.app",
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			output := parseCommitMessage(tc.commitMessage)
			t.Log(output)
			assert.Equal(t, tc.expected, output)
		})
	}
}
