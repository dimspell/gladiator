package proxy

import (
	"bytes"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/fxamacker/cbor/v2"
	"github.com/google/uuid"
)

type GlobalProxy struct {
	MaxActiveClients int
	CloseCh          chan struct{}

	Games    map[string]*Game
	mtxGames sync.RWMutex

	Connections   map[string]*Client
	mtxConnection sync.RWMutex
}

type Game struct {
	HostUserID    string
	HostIPAddress string
}

func (p *GlobalProxy) Run(bindIP string) error {
	tcpListener, err := net.Listen("tcp", net.JoinHostPort(bindIP, "6115"))
	if err != nil {
		return err
	}
	go func() {
		// p.CloseCh <- struct{}{}
		_ = tcpListener.Close()
	}()

	for {
		_, err := tcpListener.Accept()
		if err != nil {
			slog.Warn("Not accepted connection", "err", err.Error())
			continue
		}
		slog.Info("Accepted connection")

		// go p.HandleConnection(conn)
		// time.Sleep(1 * time.Second)
	}
}

func (p *GlobalProxy) HandleConnection(conn net.Conn) {
	// TODO: Write panic handler
	defer func() {
		conn.Close()
		slog.Info("Connection closed")
	}()

	p.mtxConnection.RLock()
	if len(p.Connections) >= p.MaxActiveClients {
		slog.Warn("Too many opened connections")
		return
	}
	p.mtxConnection.RUnlock()

	client, err := p.hello(conn)
	if err != nil {
		slog.Warn("Could not hello the client", "err", err.Error())
		return
	}
	p.mtxConnection.Lock()
	p.Connections[client.ID] = client
	p.mtxConnection.Unlock()
	defer func() {
		p.mtxConnection.Lock()
		delete(p.Connections, client.ID)
		p.mtxConnection.Unlock()
	}()

	h, _ := cbor.Marshal(Message{
		Command: "Hello",
		From:    "World",
		To:      "Monde",
	})
	conn.Write(h)

	for {
		buf := make([]byte, 512)
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		var message Message
		if err := cbor.Unmarshal(buf[:n], &message); err != nil {
			return
		}

		fmt.Println(message)
	}
}

type Message struct {
	Command string
	From    string
	To      string
}

type MessageTCP struct {
	Value string
}

type Client struct {
	ID   string
	User string
	Game string
}

type HelloPacket struct {
	User string
	Game string
}

func (p *GlobalProxy) hello(conn net.Conn) (*Client, error) {
	buf := make([]byte, 128)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	if n < 5 {
		return nil, fmt.Errorf("hello: invalid length: %v", buf[:n])
	}
	if !bytes.Equal([]byte("HELLO"), buf[:5]) {
		return nil, fmt.Errorf("hello: missing HELLO prefix: %v", buf[:n])
	}

	var helloPacket HelloPacket
	if err := cbor.Unmarshal(buf[5:n], &helloPacket); err != nil {
		return nil, err
	}

	id := uuid.NewString()

	slog.Info("New client",
		"id", id,
		"user", helloPacket.User,
		"game", helloPacket.Game)

	return &Client{ID: id, User: helloPacket.User, Game: helloPacket.Game}, nil
}
