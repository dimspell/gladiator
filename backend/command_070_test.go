package backend

import (
	"testing"

	"github.com/dispel-re/dispel-multi/model"
	"github.com/stretchr/testify/assert"
)

func TestRankingRequest(t *testing.T) {
	t.Run("NewRankingRequest", func(t *testing.T) {
		req := NewRankingRequest(model.ClassTypeKnight, 1000, "user", "character")
		classType, offset, user, character, err := req.Parse()

		assert.NoError(t, err)
		assert.Equal(t, model.ClassTypeKnight, classType)
		assert.Equal(t, uint32(1000), offset)
		assert.Equal(t, "user", user)
		assert.Equal(t, "character", character)
	})

	t.Run("First page for warrior", func(t *testing.T) {
		packet := []byte{
			255, 70, // Command code
			27, 0, // Packet length
			1, 0, 0, 0, // Class type
			0, 0, 0, 0, // Offset
			117, 115, 101, 114, 0, // User name
			99, 104, 97, 114, 97, 99, 116, 101, 114, 0, // Character name
		}
		req := RankingRequest(packet[4:])
		classType, offset, user, character, err := req.Parse()

		assert.NoError(t, err)
		assert.Equal(t, model.ClassTypeWarrior, classType)
		assert.Equal(t, uint32(0), offset)
		assert.Equal(t, "user", user)
		assert.Equal(t, "character", character)
	})

	t.Run("Second page for mage", func(t *testing.T) {
		packet := []byte{
			255, 70, // Command code
			22, 0, // Packet length
			3, 0, 0, 0, // Class type
			10, 0, 0, 0, // Offset
			117, 115, 101, 114, 0, // User name
			99, 104, 97, 114, 97, 99, 116, 101, 114, 0, // Character name
		}
		req := RankingRequest(packet[4:])
		classType, offset, user, character, err := req.Parse()

		assert.NoError(t, err)
		assert.Equal(t, model.ClassTypeMage, classType)
		assert.Equal(t, uint32(10), offset)
		assert.Equal(t, "user", user)
		assert.Equal(t, "character", character)
	})
}
