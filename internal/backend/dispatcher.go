package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
)

func (b *Backend) handshake(conn net.Conn) (*bsession.Session, error) {
	// Ping (single byte - [0x01])
	{
		buf := make([]byte, 1)
		_, err := conn.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("error reading: %s", err)
		}

		if buf[0] != byte(1) {
			return nil, fmt.Errorf("incorrect ping")
		}
	}

	session := b.AddSession(conn)

	// Command 255 30 aka 0x1eff
	{
		buf := make([]byte, 64)
		n, err := conn.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("error reading: %s", err)
		}

		// Reply with 255 30 aka 0x1eff
		if err := b.HandleClientHostAndUsername(session, buf[4:n]); err != nil {
			return nil, err
		}
	}

	// Command 255 6 aka 0x06ff
	{
		buf := make([]byte, 24)
		n, err := conn.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("error reading: %s", err)
		}
		if err := b.HandleAuthorizationHandshake(session, buf[4:n]); err != nil {
			return nil, err
		}
	}

	return session, nil
}

func (b *Backend) handleCommands(ctx context.Context, session *bsession.Session) error {
	buf := make([]byte, 1024)
	n, err := session.Conn.Read(buf)
	if err != nil {
		return err
	}
	packets := splitMultiPacket(buf[:n])

	for _, data := range packets {
		if len(data) < 4 {
			continue
		}
		if data[0] != 255 {
			continue
		}

		pt := packet.PacketType(data[1])
		if logger.PacketLogger != nil {
			logger.PacketLogger.Debug("Recv",
				"packetType", pt,
				"bytes", data,
				"sessionId", session.ID,
				"length", len(data),
			)
		}

		// TODO: Pass context further
		switch pt {
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
		}
	}

	return nil
}

func splitMultiPacket(buf []byte) [][]byte {
	if len(buf) < 4 {
		return [][]byte{buf}
	}

	var packets [][]byte
	var offset int
	for i := 0; i < 10; i++ {
		if (offset + 4) > len(buf) {
			break
		}

		length := int(binary.LittleEndian.Uint16(buf[offset+2 : offset+4]))
		packets = append(packets, buf[offset:offset+length])
		offset += length
	}
	return packets
}
