package model

import (
	"net"
)

type Session struct {
	ID     string
	Conn   net.Conn
	UserID int64
}
