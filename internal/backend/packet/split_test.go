package packet

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplit(t *testing.T) {
	t.Run("non-compatible packet", func(t *testing.T) {
		packets := Split([]byte{1})

		assert.Equal(t, 1, len(packets))
		assert.True(t, bytes.Equal([]byte{1}, packets[0]))
	})

	t.Run("single packet", func(t *testing.T) {
		packets := Split([]byte{255, 1, 4, 0})

		assert.Equal(t, 1, len(packets))
		assert.True(t, bytes.Equal([]byte{255, 1, 4, 0}, packets[0]))
	})

	t.Run("two packets", func(t *testing.T) {
		packets := Split([]byte{
			255, 1, 8, 0, 1, 0, 0, 0,
			255, 2, 4, 0,
		})

		assert.Equal(t, 2, len(packets))
		assert.True(t, bytes.Equal([]byte{255, 1, 8, 0, 1, 0, 0, 0}, packets[0]))
		assert.True(t, bytes.Equal([]byte{255, 2, 4, 0}, packets[1]))
	})

	t.Run("three packets", func(t *testing.T) {
		packets := Split([]byte{
			255, 1, 8, 0, 1, 0, 0, 0,
			255, 2, 4, 0,
			255, 3, 6, 0, 1, 0,
		})

		assert.Equal(t, 3, len(packets))
		assert.True(t, bytes.Equal([]byte{255, 1, 8, 0, 1, 0, 0, 0}, packets[0]))
		assert.True(t, bytes.Equal([]byte{255, 2, 4, 0}, packets[1]))
		assert.True(t, bytes.Equal([]byte{255, 3, 6, 0, 1, 0}, packets[2]))
	})

	t.Run("failing packet", func(t *testing.T) {
		createAccountPacket := []byte{
			255, 42, // header = 2  (total length = 2)
			22, 0, // length = 2 (4)
			33, 78, 0, 0, // code = 4 (8)
			116, 101, 115, 116, 0, // password "test" = 5 (13)
			116, 101, 115, 116, 117, 115, 101, 114, 0, // username "testuser" = 9 (22)
			0, 0, 49, 207, 69, 0, // gibberish, likely padding to 28 characters = 6 (28)
		}

		packets := Split(createAccountPacket)
		assert.Equal(t, 22, len(packets))
	})

	t.Run("wrong packet length - oversize", func(t *testing.T) {
		createAccountPacket := []byte{
			255, 1,
			200, 200,
			116, 101, 115, 116, 0, // password "test" = 5 (13)
		}

		packets := Split(createAccountPacket)
		assert.Equal(t, 0, len(packets))
	})
}
