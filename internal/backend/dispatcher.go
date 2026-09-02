package backend

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
)

func (b *Backend) handshake(conn net.Conn) (*bsession.Session, error) {
	// Ping (single byte - [0x01])
	{
		buf := make([]byte, 1)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return nil, fmt.Errorf("error reading: %s", err)
		}

		if buf[0] != byte(1) {
			return nil, fmt.Errorf("incorrect ping")
		}
	}

	session := b.SessionManager.Add(conn)

	// Command 255 30 aka 0x1eff
	{
		buf := make([]byte, 64)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return nil, fmt.Errorf("error reading: %s", err)
		}

		// Reply with 255 30 aka 0x1eff
		if err := b.HandleClientHostAndUsername(session, buf[4:]); err != nil {
			return nil, err
		}
	}

	// Command 255 6 aka 0x06ff
	{
		// Read header first to get length, then payload. Handles both the
		// 24-byte handshake and the shorter handshake variants the game
		// may send, without hard-coding a single length.
		header := make([]byte, 4)
		if _, err := io.ReadFull(conn, header); err != nil {
			return nil, fmt.Errorf("error reading: %s", err)
		}
		if header[0] != 255 || header[1] != 6 {
			return nil, fmt.Errorf("incorrect command 6 header: %v", header)
		}
		length := int(header[2]) | int(header[3])<<8
		if length < 4 || length > 64 {
			length = 24
		}
		payloadLen := length - 4
		if payloadLen < 0 {
			payloadLen = 0
		}
		buf := make([]byte, payloadLen)
		if payloadLen > 0 {
			if _, err := io.ReadFull(conn, buf); err != nil {
				return nil, fmt.Errorf("error reading: %s", err)
			}
		}
		// Combine header+payload for handler (handler expects data[4:] style, so pass full)
		full := append(header, buf...)
		if err := b.HandleAuthorizationHandshake(session, full[4:]); err != nil {
			// The game may send a shorter handshake variant instead.
			// Log at debug and continue; the handler acks short payloads.
			slog.Debug("handshake 0x06FF: primary handler failed, accepting short handshake", logging.Error(err))
		}
	}

	return session, nil
}

func (b *Backend) handleCommands(ctx context.Context, session *bsession.Session) error {
	// Refresh the read deadline each loop so an idle (dead) connection is
	// detected. 90s covers ~3 missed client ping intervals before we give up.
	if tcpConn, ok := session.Conn.(*net.TCPConn); ok {
		_ = tcpConn.SetReadDeadline(time.Now().Add(90 * time.Second))
	}

	buf := make([]byte, 1024)
	n, err := session.Conn.Read(buf)
	if err != nil {
		return err
	}
	packets := packet.Split(buf[:n])

	for _, data := range packets {
		if len(data) < 4 {
			continue
		}
		if data[0] != 255 {
			continue
		}

		// 4-byte lobby tokens (out-of-band signals, no command handler)
		if len(data) == 4 && data[1] == 0x0B && data[2] == 0x40 && data[3] == 0xBF {
			slog.Debug("lobby abort", "session", session.ID)
			continue
		}
		if len(data) == 4 && data[1] == 0x09 && data[2] == 0x40 && data[3] == 0xBF {
			slog.Debug("lobby refresh", "session", session.ID)
			continue
		}

		code := packet.Code(data[1])
		slog.Debug("Recv", "code", code, "bytes", data, "session_id", session.ID)

		switch code {
		case packet.CreateNewAccount:
			if err := b.HandleCreateNewAccount(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.ClientAuthentication:
			if err := b.HandleClientAuthentication(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.ListChannels:
			if err := b.HandleListChannels(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.SelectedChannel:
			if err := b.HandleSelectChannel(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.SendLobbyMessage:
			if err := b.HandleSendLobbyMessage(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.CreateGame:
			if err := b.HandleCreateGame(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.ListGames:
			if err := b.HandleListGames(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.SelectGame:
			if err := b.HandleSelectGame(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.JoinGame:
			if err := b.HandleJoinGame(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.ShowRanking:
			if err := b.HandleShowRanking(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.UpdateCharacterInventory:
			if err := b.HandleUpdateCharacterInventory(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.GetCharacters:
			if err := b.HandleGetCharacters(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.DeleteCharacter:
			if err := b.HandleDeleteCharacter(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.GetCharacterInventory:
			if err := b.HandleGetCharacterInventory(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.GetCharacterSpells:
			if err := b.HandleGetCharacterSpells(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.UpdateCharacterSpells:
			if err := b.HandleUpdateCharacterSpells(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.SelectCharacter:
			if err := b.HandleSelectCharacter(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.CreateCharacter:
			if err := b.HandleCreateCharacter(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.UpdateCharacterStats:
			if err := b.HandleUpdateCharacterStats(ctx, session, data[4:]); err != nil {
				return err
			}
		case packet.PingClockTime:
			if err := b.HandlePing(ctx, session, data[4:]); err != nil {
				return err
			}
		}
	}

	return nil
}
