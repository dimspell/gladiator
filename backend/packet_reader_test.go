package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPacketReader_ReadString(t *testing.T) {
	t.Run("Single string", func(t *testing.T) {
		rd := NewPacketReader([]byte{'t', 'e', 's', 't', nullTerminator})

		actual, err := rd.ReadString()
		assert.NoError(t, err)
		assert.Equal(t, "test", actual)
	})

	t.Run("Two strings", func(t *testing.T) {
		rd := NewPacketReader(
			[]byte{
				'h', 'e', 'l', 'l', 'o', nullTerminator,
				'w', 'o', 'r', 'l', 'd', nullTerminator,
			},
		)

		hello, err := rd.ReadString()
		assert.NoError(t, err)
		assert.Equal(t, "hello", hello)

		world, err := rd.ReadString()
		assert.NoError(t, err)
		assert.Equal(t, "world", world)
	})
}
