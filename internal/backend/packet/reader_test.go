package packet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReader(t *testing.T) {
	t.Run("Single string", func(t *testing.T) {
		rd := NewReader([]byte{'t', 'e', 's', 't', nullTerminator})

		actual, err := rd.ReadString()
		assert.NoError(t, err)
		assert.Equal(t, "test", actual)
	})

	t.Run("Two strings", func(t *testing.T) {
		rd := NewReader(
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

	t.Run("Empty string", func(t *testing.T) {
		rd := NewReader([]byte{nullTerminator})

		actual, err := rd.ReadString()
		assert.NoError(t, err)
		assert.Equal(t, "", actual)
	})

	t.Run("Uint8", func(t *testing.T) {
		rd := NewReader([]byte{0x01})

		actual, err := rd.ReadUint8()
		assert.NoError(t, err)
		assert.Equal(t, uint8(0x01), actual)
	})

	t.Run("Uint16", func(t *testing.T) {
		rd := NewReader([]byte{0x01, 0x02})

		actual, err := rd.ReadUint16()
		assert.NoError(t, err)
		assert.Equal(t, uint16(0x0201), actual)
	})

	t.Run("Uint32", func(t *testing.T) {
		rd := NewReader([]byte{0x01, 0x02, 0x03, 0x04})

		actual, err := rd.ReadUint32()
		assert.NoError(t, err)
		assert.Equal(t, uint32(0x04030201), actual)
	})

	t.Run("Read rest bytes", func(t *testing.T) {
		rd := NewReader(
			[]byte{
				'h', 'e', 'l', 'l', 'o', nullTerminator,
				0x01, 0x02, 0x03, 0x04,
			},
		)
		{
			hello, err := rd.ReadString()
			assert.NoError(t, err)
			assert.Equal(t, "hello", hello)
		}
		{
			rest, err := rd.ReadRestBytes()
			assert.NoError(t, err)
			assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, rest)
		}
	})
}
