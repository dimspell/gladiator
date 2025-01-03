package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet/command"
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

	for _, packet := range packets {
		if len(packet) < 4 {
			continue
		}
		if packet[0] != 255 {
			continue
		}

		pt := command.PacketType(packet[1])
		if logger.PacketLogger != nil {
			logger.PacketLogger.Debug("Recv",
				"packetType", pt,
				"bytes", packet,
				"sessionId", session.ID,
				"length", len(packet),
			)
		}

		// TODO: Pass context further
		switch pt {
		case command.CreateNewAccount:
			if err := b.HandleCreateNewAccount(session, packet[4:]); err != nil {
				return err
			}
		case command.ClientAuthentication:
			if err := b.HandleClientAuthentication(session, packet[4:]); err != nil {
				return err
			}
		case command.ListChannels:
			if err := b.HandleListChannels(session, packet[4:]); err != nil {
				return err
			}
		case command.SelectedChannel:
			if err := b.HandleSelectChannel(session, packet[4:]); err != nil {
				return err
			}
		case command.SendLobbyMessage:
			if err := b.HandleSendLobbyMessage(session, packet[4:]); err != nil {
				return err
			}
		case command.CreateGame:
			if err := b.HandleCreateGame(session, packet[4:]); err != nil {
				return err
			}
		case command.ListGames:
			if err := b.HandleListGames(session, packet[4:]); err != nil {
				return err
			}
		case command.SelectGame:
			if err := b.HandleSelectGame(session, packet[4:]); err != nil {
				return err
			}
		case command.JoinGame:
			if err := b.HandleJoinGame(session, packet[4:]); err != nil {
				return err
			}
		case command.ShowRanking:
			if err := b.HandleShowRanking(session, packet[4:]); err != nil {
				return err
			}
		case command.UpdateCharacterInventory:
			if err := b.HandleUpdateCharacterInventory(session, packet[4:]); err != nil {
				return err
			}
		case command.GetCharacters:
			if err := b.HandleGetCharacters(session, packet[4:]); err != nil {
				return err
			}
		case command.DeleteCharacter:
			if err := b.HandleDeleteCharacter(session, packet[4:]); err != nil {
				return err
			}
		case command.GetCharacterInventory:
			if err := b.HandleGetCharacterInventory(session, packet[4:]); err != nil {
				return err
			}
		case command.GetCharacterSpells:
			if err := b.HandleGetCharacterSpells(session, packet[4:]); err != nil {
				return err
			}
		case command.UpdateCharacterSpells:
			if err := b.HandleUpdateCharacterSpells(session, packet[4:]); err != nil {
				return err
			}
		case command.SelectCharacter:
			if err := b.HandleSelectCharacter(session, packet[4:]); err != nil {
				return err
			}
		case command.CreateCharacter:
			if err := b.HandleCreateCharacter(session, packet[4:]); err != nil {
				return err
			}
		case command.UpdateCharacterStats:
			if err := b.HandleUpdateCharacterStats(session, packet[4:]); err != nil {
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
