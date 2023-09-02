package model

import (
	"net"
)

type Session struct {
	ID        string
	User      *User
	Character *Character
	Conn      net.Conn
}

func (s *Session) LoggedIn() bool {
	return s.Character != nil
}

type User struct {
	UserName   string
	Characters []Character
}

type Character struct {
	CharacterName string
	Slot          int
	Info          CharacterInfo
	Inventory     CharacterInventory
	Spells        []byte
}
