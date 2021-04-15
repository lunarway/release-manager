package kubernetes

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMisingLabelsIsNotCorrectlyAnnotated(t *testing.T) {

	annotations := map[string]string{}
	annotations["lunarway.com/controlled-by-release-manager"] = "true"
	correctlyAnnotated := isCorrectlyAnnotated(annotations)

	assert.False(t, correctlyAnnotated)
}

func TestAllLabelsSetIsCorrectlyAnnotated(t *testing.T) {

	annotations := map[string]string{}
	annotations["lunarway.com/controlled-by-release-manager"] = "true"
	annotations["lunarway.com/artifact-id"] = "1"
	annotations["lunarway.com/author"] = "platon"
	correctlyAnnotated := isCorrectlyAnnotated(annotations)

	assert.True(t, correctlyAnnotated)
}
