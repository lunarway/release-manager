package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsCorrectlyAnnotated(t *testing.T) {
	tt := []struct {
		name       string
		controlled string
		artifactID string
		author     string
		correct    bool
	}{
		{
			name:       "only controlled",
			controlled: "true",
			artifactID: "",
			author:     "",
			correct:    false,
		},
		{
			name:       "all good",
			controlled: "true",
			artifactID: "1",
			author:     "platon",
			correct:    true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			correct := isCorrectlyAnnotated(map[string]string{
				controlledAnnotationKey: tc.controlled,
				artifactIDAnnotationKey: tc.artifactID,
				authorAnnotationKey:     tc.author,
			})

			assert.Equal(t, tc.correct, correct)
		})
	}
}
