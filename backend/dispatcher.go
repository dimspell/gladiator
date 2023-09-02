package backend

import (
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"

	"github.com/dispel-re/dispel-multi/model"
)

type PacketType byte

const (
	AuthorizationHandshake   PacketType = 6   // 0x6ff
	ListGames                PacketType = 9   // 0x9ff
	ListChannels             PacketType = 11  // 0xbff
	SelectedChannel          PacketType = 12  // 0xcff
	SendLobbyMessage         PacketType = 14  // 0xeff
	ReceiveMessage           PacketType = 15  // 0xfff
	PingClockTime            PacketType = 21  // 0x15ff
	CreateGame               PacketType = 28  // 0x1cff
	ClientHostAndUsername    PacketType = 30  // 0x1eff
	JoinGame                 PacketType = 34  // 0x22ff
	ClientAuthentication     PacketType = 41  // 0x29ff
	CreateNewAccount         PacketType = 42  // 0x2aff
	UpdateCharacterInventory PacketType = 44  // 0x2cff
	GetCharacters            PacketType = 60  // 0x3cff
	DeleteCharacter          PacketType = 61  // 0x3dff
	GetCharacterInventory    PacketType = 68  // 0x44ff
	SelectGame               PacketType = 69  // 0x45ff
	ShowRanking              PacketType = 70  // 0x46ff
	ChangeHost               PacketType = 71  // 0x47ff
	GetCharacterSpells       PacketType = 72  // 0x48ff
	UpdateCharacterSpells    PacketType = 73  // 0x49ff
	SelectCharacter          PacketType = 76  // 0x4cff
	CreateCharacter          PacketType = 92  // 0x5cff
	UpdateCharacterStats     PacketType = 108 // 0x6cff
)

func (b *Backend) Listen(backendAddr string) {
	// Listen for incoming connections.
	l, err := net.Listen("tcp4", backendAddr)
	if err != nil {
		slog.Error("Could not start listening on port 6112", "err", err)
		os.Exit(1)
	}

	// Close the listener when the application closes.
	defer l.Close()

	slog.Info("Listening for new connections...")
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			continue
		}
		slog.Info("Accepted connection",
			slog.String("remoteAddr", conn.RemoteAddr().String()),
			slog.String("localAddr", conn.LocalAddr().String()),
		)

		// Handle connections in a new goroutine.
		// go handleRequest(connPort, conn)
		go func() {
			if err := b.handleClient(conn); err != nil {
				slog.Warn("Communication with client has failed",
					"err", err)
			}
		}()
	}
}

func (b *Backend) handleClient(conn net.Conn) error {
	session, err := b.handshake(conn)
	if err != nil {
		conn.Close()
		if err == io.EOF {
			return nil
		}
		return err
	}
	defer b.CloseSession(session)

	for {
		if err := b.handleCommands(session); err != nil {
			return err
		}
	}
}

func (b *Backend) handshake(conn net.Conn) (*model.Session, error) {
	// Hello World
	buf := make([]byte, 1)
	{
		_, err := conn.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("error reading: %s", err)
		}
	}
	if buf[0] != byte(1) {
		return nil, fmt.Errorf("incorrect ping")
	}

	session := b.NewSession(conn)

	// Command 255 30 aka 0x1eff
	buf = make([]byte, 64)
	{
		n, err := conn.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("error reading: %s", err)
		}
		fmt.Println(string(buf[:n]), n, buf[:n])

		// Reply with 255 30 aka 0x1eff
		if err := b.HandleClientHostAndUsername(session, buf[:n]); err != nil {
			return nil, err
		}
	}

	// Command 255 6 aka 0x06ff
	buf = make([]byte, 24)
	{
		n, err := conn.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("error reading: %s", err)
		}
		if err := b.HandleAuthorizationHandshake(session, buf[:n]); err != nil {
			return nil, err
		}
	}

	return session, nil
}

func (b *Backend) handleCommands(session *model.Session) error {
	buf := make([]byte, 1024)
	if _, err := session.Conn.Read(buf); err != nil {
		return err
	}
	packets := splitMultiPacket(buf)

	for _, packet := range packets {
		pt := PacketType(packet[1])

		switch pt {
		case CreateNewAccount:
			break
		case ClientAuthentication:
			break
		case ListChannels:
			break
		case SelectedChannel:
			break
		case CreateGame:
			break
		case ListGames:
			break
		case SelectGame:
			break
		case JoinGame:
			break
		case ShowRanking:
			break
		}
	}
	return nil
}

func splitMultiPacket(buf []byte) [][]byte {
	if len(buf) < 4 {
		return [][]byte{buf}
	}

	packets := [][]byte{}
	var offset int
	for {
		if (offset + 4) > len(buf) {
			break
		}

		length := int(binary.LittleEndian.Uint16(buf[offset+2 : offset+4]))
		packets = append(packets, buf[offset:offset+length])
		offset += length
	}
	return packets
}

func (b *Backend) Send(conn net.Conn, packetType PacketType, payload []byte) error {
	_, err := conn.Write(b.EncodePacket(packetType, payload))
	return err
}

func (b *Backend) EncodePacket(packetType PacketType, payload []byte) []byte {
	length := len(payload) + 4
	packet := make([]byte, length)

	// Header
	packet[0] = 255
	packet[1] = byte(packetType)
	binary.LittleEndian.PutUint16(packet[2:4], uint16(length))

	// Data
	copy(packet[4:], payload)

	return packet
}
