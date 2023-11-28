package dto

import (
	"github.com/dispel-re/dispel-multi/model"
)

type Character struct {
	UserId        int64  `json:"userId"`
	UserName      string `json:"userName"`
	CharacterId   int64  `json:"characterId"`
	CharacterName string `json:"characterName"`

	CharacterInfo model.CharacterInfo `json:"characterInfo,omitempty"`
}
