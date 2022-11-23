package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReleaseSucceeded(t *testing.T) {
	t.Run("type is correct", func(t *testing.T) {
		sut := ReleaseSucceeded{}

		actual := sut.Type()

		assert.Equal(t, "releaseSucceeded", actual)
	})
}
