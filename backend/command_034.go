package backend

import "github.com/dispel-re/dispel-multi/model"

// HandleJoinGame handles 0x22ff (255-34) command
func (b *Backend) HandleJoinGame(session *model.Session, req JoinGameRequest) error {
	// gameRoom := b.DB.GameRooms()[0]
	var gameRoom model.GameRoom
	return b.Send(session.Conn, JoinGame, gameRoom.Details())
}

type JoinGameRequest []byte

// String of the room name
// /dispatcher.go:261 msg="Sent packet" packetType=69 data="\xffE\b\x00\x00\x00\x00\x00"
// /dispatcher.go:151 msg="Handle packet" packetType=69 packet="\xffE\x06\x00\x01\x00"
// /dispatcher.go:261 msg="Sent packet" packetType=69 data="\xffE\b\x00\x00\x00\x00\x00"
// /dispatcher.go:151 msg="Handle packet" packetType=34 packet="\xff\"\a\x00\x01\x00\x19"
// /dispatcher.go:261 msg="Sent packet" packetType=34 data="\xff\"\b\x00\x00\x00\x00\x00"

// /dispatcher.go:264 msg="Sent packet" packetType=69 data="/0UIAAAAAAA="
// /dispatcher.go:152 msg="Handle packet" packetType=69 packet=/0UGAAEA
// /dispatcher.go:264 msg="Sent packet" packetType=69 data="/0UIAAAAAAA="
// /dispatcher.go:152 msg="Handle packet" packetType=34 packet="/yIHAAEAGQ=="
// /dispatcher.go:264 msg="Sent packet" packetType=34 data="/yIIAAAAAAA="
