package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReleaseEvent(t *testing.T) {
	t.Run("type is correct", func(t *testing.T) {
		sut := ReleasedEvent{}

		actual := sut.Type()

		assert.Equal(t, "released", actual)
	})
}
