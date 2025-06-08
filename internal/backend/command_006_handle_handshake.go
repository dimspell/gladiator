package backend

import (
	"fmt"
	"log/slog"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
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
func (b *Backend) HandleAuthorizationHandshake(session *bsession.Session, req AuthorizationHandshakeRequest) error {
	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", logging.Error(err))
		return nil
	}

	if string(data.AuthKey) != "68XIPSID" {
		if err := session.SendToGame(packet.AuthorizationHandshake, []byte{0, 0, 0, 0}); err != nil {
			return err
		}

		// Returned only for any fake clients
		return fmt.Errorf("packet-6: wrong auth key: %q", data.AuthKey)
	}

	if data.VersionNumber != 3 {
		if err := session.SendToGame(packet.AuthorizationHandshake, []byte{0, 0, 0, 0}); err != nil {
			return err
		}

		return fmt.Errorf("packet-6: invalid version number: %d", data.VersionNumber)
	}

	return session.SendToGame(packet.AuthorizationHandshake, []byte("ENET\x00"))
}

type AuthorizationHandshakeRequest []byte

type AuthorizationHandshakeRequestData struct {
	// Authorization key. Normally it should be equal to "68XIPSID".
	AuthKey []byte

	// It seems to be always equal to 3.
	VersionNumber uint32
}

// Parse extract data from the command packet.
func (r AuthorizationHandshakeRequest) Parse() (data AuthorizationHandshakeRequestData, err error) {
	if len(r) < 12 {
		return data, fmt.Errorf("packet-6: invalid length: %d", len(r))
	}

	rd := packet.NewReader(r)
	data.AuthKey, err = rd.ReadNBytes(8)
	if err != nil {
		return data, fmt.Errorf("packet-6: malformed auth key: %w", err)
	}
	data.VersionNumber, err = rd.ReadUint32()
	if err != nil {
		return data, fmt.Errorf("packet-6: malformed version number: %w", err)
	}
	return data, rd.Close()
}
