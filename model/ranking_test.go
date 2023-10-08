package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRanking(t *testing.T) {
	ranking := Ranking{
		Players: []RankingPosition{
			{
				Rank:          1,
				Points:        200,
				Username:      "User1",
				CharacterName: "Warrior",
			},
			{
				Rank:          2,
				Points:        150,
				Username:      "Current",
				CharacterName: "Mage",
			},
		},
		CurrentPlayer: RankingPosition{
			Rank:          2,
			Points:        150,
			Username:      "Current",
			CharacterName: "Mage",
		},
	}

	assert.True(t, bytes.Equal(
		[]byte{
			2, 0, 0, 0, // Number of characters

			1, 0, 0, 0, // Ranking position = 1
			200, 0, 0, 0, // Points = 200
			85, 115, 101, 114, 49, 0, // Username = "User1"
			87, 97, 114, 114, 105, 111, 114, 0, // Character name = "Warrior"

			2, 0, 0, 0, // Ranking position = 2
			150, 0, 0, 0, // Points = 150
			67, 117, 114, 114, 101, 110, 116, 0, // Username = "Current"
			77, 97, 103, 101, 0, // Character name = "Mage"

			2, 0, 0, 0, // Ranking position = 2
			150, 0, 0, 0, // Points = 150
			67, 117, 114, 114, 101, 110, 116, 0, // Username = "Current"
			77, 97, 103, 101, 0, // Character name = "Mage"
		},
		ranking.ToBytes(),
	))
}
