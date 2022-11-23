package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReleaseFailed(t *testing.T) {
	t.Run("type is correct", func(t *testing.T) {
		sut := ReleaseFailed{}

		actual := sut.Type()

		assert.Equal(t, "releaseFailed", actual)
	})
}
