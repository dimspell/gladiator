package model

import (
	"net"
)

type Session struct {
	ID          string
	Conn        net.Conn
	UserID      int64
	CharacterID int64
	Username    string
}
