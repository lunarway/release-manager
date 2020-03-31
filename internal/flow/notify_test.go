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
			name:          "exact values",
			commitMessage: "[env/service-name] release master-1234567890-1234567890 by test@lunar.app",
			expected: FluxReleaseMessage{
				Environment: "env",
				Service:     "service-name",
				ArtifactID:  "master-1234567890-1234567890",
				GitAuthor:   "test@lunar.app",
			},
			err: nil,
		},
		{
			name:          "only three values match",
			commitMessage: "[env/service-name] release master-1234567890-1234567890",
			expected:      FluxReleaseMessage{},
			err:           errors.New("lenght of matches not as expected"),
		},
		{
			name:          "random commit message",
			commitMessage: "test test test test",
			expected:      FluxReleaseMessage{},
			err:           errors.New("lenght of matches not as expected"),
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
