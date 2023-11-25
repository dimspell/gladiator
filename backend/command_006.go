package backend

import (
	"encoding/binary"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleAuthorizationHandshake handles 0x6ff (255-6) command.
//
// This command is called from the game client during initial handshake, after
// player clicked on the "Play" button and the game server previously responded
// on command 255-30.
//
// It expects to receive an authorization key "68XIPSID" (note: not a null
// terminated string) from the game client. If the key matches, then the game
// server is going to respond with "ENET" (also a null-terminated string).
//
// When the game client will receive the response on the 255-6 command, it is
// going to display a login screen, asking user to create a new account or sign
// in using with already existing credentials.
func (b *Backend) HandleAuthorizationHandshake(session *model.Session, req AuthorizationHandshakeRequest) error {
	data, err := req.Parse()
	if err != nil {
		return fmt.Errorf("packet-6: %w", err)
	}
	if data.AuthKey != "68XIPSID" {
		if err := b.Send(session.Conn, AuthorizationHandshake, []byte{0, 0, 0, 0}); err != nil {
			return err
		}

		// Returned only for any fake clients
		return fmt.Errorf("packet-6: wrong auth key: %q", data.AuthKey)
	}

	return b.Send(session.Conn, AuthorizationHandshake, []byte("ENET\x00"))
}

type AuthorizationHandshakeRequest []byte

type AuthorizationHandshakeRequestData struct {
	// Authorization key. Normally it should be equal to "68XIPSID".
	AuthKey string

	// TODO: Recognise what kind of integer does it store. It seems to be always equal to 3.
	UnknownCounter uint32
}

// Parse extract data from the command packet.
func (r AuthorizationHandshakeRequest) Parse() (data AuthorizationHandshakeRequestData, err error) {
	if len(r) < 12 {
		return data, fmt.Errorf("malformed packet")
	}

	data.AuthKey = string(r[:8])
	data.UnknownCounter = binary.LittleEndian.Uint32(r[8:12])
	return data, err
}
