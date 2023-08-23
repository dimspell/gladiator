package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"

	"github.com/dispel-re/dispel-multi/database/memory"
	"github.com/dispel-re/dispel-multi/model"
)

type Backend struct {
	DB *memory.Memory

	Sessions map[string]*model.Session
}

func NewBackend(db *memory.Memory) *Backend {
	return nil
}

func (b *Backend) Shutdown(ctx context.Context) {
	// Close all open connections
	for _, session := range b.Sessions {
		session.Conn.Close()
	}

	// TODO: Send a system message "(system) The server is going to close in less than 30 seconds"
	// TODO: Send a packet to trigger stats saving
	// TODO: Send a system message "(system): Your stats were saving, your game client might close in next 10 seconds"
	// TODO: Send a packet to close the connection (malformed 255-21?)
}

func (b *Backend) NewSession(conn net.Conn) *model.Session {
	session := &model.Session{Conn: conn}
	b.Sessions["id"] = session
	return session
}

func (b *Backend) CloseSession(session *model.Session) error {
	// TODO: wrap all errors
	_, ok := b.Sessions["id"]
	if ok {
		delete(b.Sessions, "id")
	}

	if session.Conn != nil {
		session.Conn.Close()
	}
	return nil
}

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
	session, err := b.Handshake(conn)
	if err != nil {
		conn.Close()
		if err == io.EOF {
			return nil
		}
		return err
	}
	defer b.CloseSession(session)

	for {
		if err := b.HandleCommands(session); err != nil {
			return err
		}
	}
}

func (b *Backend) Handshake(conn net.Conn) (*model.Session, error) {
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

func (b *Backend) HandleCommands(session *model.Session) error {
	var buf []byte
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
	todo := [][]byte{
		buf,
	}
	return todo
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
