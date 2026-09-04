package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigChangedEvent(t *testing.T) {
	t.Run("type is correct", func(t *testing.T) {
		sut := ConfigChangedEvent{}

		actual := sut.Type()

		assert.Equal(t, "config_changed", actual)
	})

	t.Run("marshal and unmarshal round-trips the SHA", func(t *testing.T) {
		sut := ConfigChangedEvent{SHA: "abc123"}

		data, err := sut.Marshal()
		require.NoError(t, err)

		var got ConfigChangedEvent
		err = got.Unmarshal(data)
		require.NoError(t, err)

		assert.Equal(t, sut, got)
	})
}
