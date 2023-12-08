package backend

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_splitMultiPacket(t *testing.T) {
	t.Run("non-compatible packet", func(t *testing.T) {
		packets := splitMultiPacket([]byte{1})

		assert.Equal(t, 1, len(packets))
		assert.True(t, bytes.Equal([]byte{1}, packets[0]))
	})

	t.Run("single packet", func(t *testing.T) {
		packets := splitMultiPacket([]byte{255, 1, 4, 0})

		assert.Equal(t, 1, len(packets))
		assert.True(t, bytes.Equal([]byte{255, 1, 4, 0}, packets[0]))
	})

	t.Run("two packets", func(t *testing.T) {
		packets := splitMultiPacket([]byte{
			255, 1, 8, 0, 1, 0, 0, 0,
			255, 2, 4, 0,
		})

		assert.Equal(t, 2, len(packets))
		assert.True(t, bytes.Equal([]byte{255, 1, 8, 0, 1, 0, 0, 0}, packets[0]))
		assert.True(t, bytes.Equal([]byte{255, 2, 4, 0}, packets[1]))
	})

	t.Run("three packets", func(t *testing.T) {
		packets := splitMultiPacket([]byte{
			255, 1, 8, 0, 1, 0, 0, 0,
			255, 2, 4, 0,
			255, 3, 6, 0, 1, 0,
		})

		assert.Equal(t, 3, len(packets))
		assert.True(t, bytes.Equal([]byte{255, 1, 8, 0, 1, 0, 0, 0}, packets[0]))
		assert.True(t, bytes.Equal([]byte{255, 2, 4, 0}, packets[1]))
		assert.True(t, bytes.Equal([]byte{255, 3, 6, 0, 1, 0}, packets[2]))
	})
}
