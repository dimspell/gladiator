package backend

import (
	"testing"

	"github.com/dispel-re/dispel-multi/model"
	"github.com/stretchr/testify/assert"
)

func TestRankingRequest(t *testing.T) {
	t.Run("NewRankingRequest", func(t *testing.T) {
		req := NewRankingRequest(RankingRequestData{
			ClassType:     model.ClassTypeKnight,
			Offset:        1000,
			Username:      "user",
			CharacterName: "character",
		})
		data, err := req.Parse()

		assert.NoError(t, err)
		assert.Equal(t, model.ClassTypeKnight, data.ClassType)
		assert.Equal(t, uint32(1000), data.Offset)
		assert.Equal(t, "user", data.Username)
		assert.Equal(t, "character", data.CharacterName)
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
		data, err := req.Parse()

		assert.NoError(t, err)
		assert.Equal(t, model.ClassTypeWarrior, data.ClassType)
		assert.Equal(t, uint32(0), data.Offset)
		assert.Equal(t, "user", data.Username)
		assert.Equal(t, "character", data.CharacterName)
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
		data, err := req.Parse()

		assert.NoError(t, err)
		assert.Equal(t, model.ClassTypeMage, data.ClassType)
		assert.Equal(t, uint32(10), data.Offset)
		assert.Equal(t, "user", data.Username)
		assert.Equal(t, "character", data.CharacterName)
	})
}
