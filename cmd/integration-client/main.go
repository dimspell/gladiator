// Command integration-client is a minimal, deterministic game client used by the
// multi-docker integration tests. It speaks the real backend wire protocol over
// TCP :6112 (handshake + lobby/room phase) and then performs a real
// game-packet exchange over UDP :6113 / TCP :6114 with its peer.
//
// For the LAN proxy the game traffic is direct peer-to-peer: the host listens on
// its own MY_IP and the guest sends to PEER_IP. The host learns the guest's
// address from the source of the incoming handshake packet and replies on it, so a
// full bidirectional exchange is verified without any relay/p2p proxy in the path.
//
// Success is reported by printing GAME_PACKET_OK and exiting 0. Any failure
// prints a diagnostic and exits 1 so the test harness can fail fast.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	opHostAndUsername = 30 // 0x1eff
	opAuthHandshake   = 6  // 0x6ff
	opClientAuth      = 41 // 0x29ff
	opSelectCharacter = 76 // 0x4cff
	opGetCharInventory = 68 // 0x44ff
	opCreateGame      = 28 // 0x1cff
	opListGames       = 9  // 0x9ff
	opSelectGame      = 69 // 0x45ff
	opJoinGame        = 34 // 0x22ff

	gamePortUDP = "6113"
	gamePortTCP = "6114"

	handshakeMagic = "\x1a\x00\x02\x00" // {26,0,2,0}
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "MOCKCLIENT_ERROR:", err)
		os.Exit(1)
	}
	fmt.Println("GAME_PACKET_OK")
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func run() error {
	backendAddr := env("BACKEND_ADDR", "127.0.0.1:6112")
	role := env("ROLE", "guest") // "host" or "guest"
	username := env("USERNAME", "tester")
	room := env("ROOM", "room")
	myIP := env("MY_IP", "127.0.0.1")
	peerIP := env("PEER_IP", "")
	timeout := 60 * time.Second
	if v := env("TIMEOUT_SECONDS", ""); v != "" {
		if sec, err := strconv.Atoi(v); err == nil {
			timeout = time.Duration(sec) * time.Second
		}
	}

	conn, err := net.DialTimeout("tcp", backendAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("dial backend %s: %w", backendAddr, err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))

	// Drain backend responses so its write buffer never blocks.
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := conn.Read(buf); err != nil {
				return
			}
		}
	}()

	if err := handshake(conn); err != nil {
		return fmt.Errorf("handshake: %w", err)
	}
	if err := clientAuth(conn, username); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	if err := selectCharacter(conn, username); err != nil {
		return fmt.Errorf("select character: %w", err)
	}
	// Opcode 68 triggers InitObserver -> JoinLobby, which registers this
	// user in the console lobby so CreateRoom/JoinRoom can find the session.
	// The backend only replies if the (real) inventory is exactly 207 bytes,
	// which our mock user has none of -- so we send it and do NOT wait
	// for a response. The drain goroutine consumes anything sent.
	if err := triggerObserver(conn, username); err != nil {
		return fmt.Errorf("trigger observer: %w", err)
	}
	// Let the console finish registering the lobby session before we
	// create/join the room (the registration happens just after the
	// JoinedLobby reply on the console side).
	time.Sleep(500 * time.Millisecond)

	switch role {
	case "host":
		if err := hostRoom(conn, room); err != nil {
			return fmt.Errorf("host room: %w", err)
		}
	case "guest":
		if peerIP == "" {
			return fmt.Errorf("guest requires PEER_IP (host game address)")
		}
		if err := guestRoom(conn, room); err != nil {
			return fmt.Errorf("guest room: %w", err)
		}
	default:
		return fmt.Errorf("unknown ROLE %q", role)
	}

	if err := exchange(myIP, peerIP, role, timeout); err != nil {
		return fmt.Errorf("game exchange: %w", err)
	}
	return nil
}

// handshake performs the 3-step TCP handshake expected by backend.handleClient.
// The backend reads 1 byte (ping), then a 64-byte frame, then a 24-byte
// frame using non-looping conn.Read, so we send each frame as its own write
// and pace them slightly. This lets the backend's reads consume each frame
// completely before the next one is transmitted, avoiding a short-read race.
func handshake(conn net.Conn) error {
	// 1) ping byte
	if _, err := conn.Write([]byte{1}); err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)
	// 2) command 255-30: 64-byte frame, 60-byte payload (two null-terminated strings)
	hostAndUser := encodePacket(opHostAndUsername, pad([]byte("host\x00user\x00"), 60))
	if len(hostAndUser) != 64 {
		return fmt.Errorf("host/username frame must be 64 bytes, got %d", len(hostAndUser))
	}
	if _, err := conn.Write(hostAndUser); err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)
	// 3) command 255-6: 24-byte frame, 20-byte payload ("68XIPSID" + uint32(3) + pad)
	authPayload := make([]byte, 20)
	copy(authPayload, "68XIPSID")
	binary.LittleEndian.PutUint32(authPayload[8:12], 3)
	authFrame := encodePacket(opAuthHandshake, authPayload)
	if len(authFrame) != 24 {
		return fmt.Errorf("auth handshake frame must be 24 bytes, got %d", len(authFrame))
	}
	if _, err := conn.Write(authFrame); err != nil {
		return err
	}
	return nil
}

func clientAuth(conn net.Conn, username string) error {
	payload := append([]byte{2, 0, 0, 0, 't', 'e', 's', 't', 0}, []byte(username+"\x00")...)
	return writeFrame(conn, opClientAuth, payload)
}

func selectCharacter(conn net.Conn, username string) error {
	payload := []byte(username + "\x00" + username + "\x00")
	return writeFrame(conn, opSelectCharacter, payload)
}

func hostRoom(conn net.Conn, room string) error {
	// state 0 -> CreateRoom
	if err := writeFrame(conn, opCreateGame, createGamePayload(0, room)); err != nil {
		return err
	}
	// state 1 -> SetRoomReady (room becomes Ready, guest may join)
	if err := writeFrame(conn, opCreateGame, createGamePayload(1, room)); err != nil {
		return err
	}
	return nil
}

func guestRoom(conn net.Conn, room string) error {
	if err := writeFrame(conn, opListGames, nil); err != nil {
		return err
	}
	if err := writeFrame(conn, opSelectGame, []byte(room+"\x00")); err != nil {
		return err
	}
	if err := writeFrame(conn, opJoinGame, []byte(room+"\x00")); err != nil {
		return err
	}
	return nil
}

// triggerObserver sends opcode 68, which makes the backend call
// InitObserver -> JoinLobby, registering this user in the console lobby so
// that CreateRoom/JoinRoom can resolve the session. The backend only
// replies if the (real) inventory is exactly 207 bytes, which our mock
// user lacks, so we do NOT wait for a response -- the drain goroutine
// consumes anything the backend happens to send.
func triggerObserver(conn net.Conn, username string) error {
	if err := writeFrame(conn, opGetCharInventory, []byte(username+"\x00"+username+"\x00")); err != nil {
		return err
	}
	return nil
}

func createGamePayload(state uint32, room string) []byte {
	p := make([]byte, 4)
	binary.LittleEndian.PutUint32(p, state)
	p = append(p, byte(1), 0, 0, 0) // map id = 1 (valid range 0-5)
	p = append(p, []byte(room+"\x00")...)
	p = append(p, 0) // password
	return p
}

// exchange performs a bidirectional UDP + TCP game-packet exchange with the peer.
// Host listens on MY_IP; guest sends to PEER_IP. The host learns the guest's
// address from the incoming handshake source and replies on it.
func exchange(myIP, peerIP, role string, timeout time.Duration) error {
	type result struct {
		proto string
		err   error
	}
	results := make(chan result, 2)

	// UDP
	go func() {
		err := exchangeUDP(myIP, peerIP, role, timeout)
		results <- result{"udp", err}
	}()
	// TCP
	go func() {
		err := exchangeTCP(myIP, peerIP, role, timeout)
		results <- result{"tcp", err}
	}()

	var firstErr error
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", r.proto, r.err)
			}
			fmt.Fprintf(os.Stderr, "MOCKCLIENT_WARN: %s exchange failed: %v\n", r.proto, r.err)
		} else {
			fmt.Printf("GAME_PACKET_EXCHANGED_%s\n", strings.ToUpper(r.proto))
		}
	}
	return firstErr
}

func exchangeUDP(myIP, peerIP, role string, timeout time.Duration) error {
	payload := []byte("udp-game-packet-from-" + role)
	deadline := time.Now().Add(timeout)
	magic := []byte(handshakeMagic)

	if role == "host" {
		pc, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(myIP), Port: 6113})
		if err != nil {
			return fmt.Errorf("listen udp: %w", err)
		}
		defer pc.Close()
		pc.SetReadDeadline(deadline)

		// Receive one datagram: handshake magic + game payload.
		buf := make([]byte, 1024)
		n, remote, err := pc.ReadFromUDP(buf)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		got := buf[:n]
		if !bytes.HasPrefix(got, magic) {
			return fmt.Errorf("unexpected udp handshake: %q", string(got))
		}
		if len(got) <= len(magic) {
			return fmt.Errorf("empty udp payload: %q", string(got))
		}
		// Reply to the guest on the address it sent from.
		reply := []byte("udp-reply-from-host")
		if _, err := pc.WriteToUDP(reply, remote); err != nil {
			return fmt.Errorf("write reply: %w", err)
		}
		return nil
	}

	// guest
	remote, err := net.ResolveUDPAddr("udp", net.JoinHostPort(peerIP, gamePortUDP))
	if err != nil {
		return fmt.Errorf("resolve peer: %w", err)
	}
	pc, err := net.DialUDP("udp", nil, remote)
	if err != nil {
		return fmt.Errorf("dial peer: %w", err)
	}
	defer pc.Close()
	pc.SetWriteDeadline(deadline)
	// Send magic + payload as a single datagram.
	msg := append(append([]byte{}, magic...), payload...)
	if _, err := pc.Write(msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	pc.SetReadDeadline(deadline)
	buf := make([]byte, 1024)
	n, err := pc.Read(buf)
	if err != nil {
		return fmt.Errorf("read reply: %w", err)
	}
	if string(buf[:n]) != "udp-reply-from-host" {
		return fmt.Errorf("unexpected udp reply: %q", string(buf[:n]))
	}
	return nil
}

func exchangeTCP(myIP, peerIP, role string, timeout time.Duration) error {
	payload := []byte("tcp-game-packet-from-" + role)
	deadline := time.Now().Add(timeout)
	magic := []byte(handshakeMagic)

	if role == "host" {
		ln, err := net.Listen("tcp", net.JoinHostPort(myIP, gamePortTCP))
		if err != nil {
			return fmt.Errorf("listen tcp: %w", err)
		}
		defer ln.Close()
		ln.(*net.TCPListener).SetDeadline(deadline)
		c, err := ln.Accept()
		if err != nil {
			return fmt.Errorf("accept: %w", err)
		}
		defer c.Close()
		c.SetReadDeadline(deadline)
		buf := make([]byte, 1024)
		n, err := c.Read(buf)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		got := buf[:n]
		if !bytes.HasPrefix(got, magic) {
			return fmt.Errorf("unexpected tcp handshake: %q", string(got))
		}
		if len(got) <= len(magic) {
			return fmt.Errorf("empty tcp payload: %q", string(got))
		}
		c.SetWriteDeadline(deadline)
		if _, err := c.Write([]byte("tcp-reply-from-host")); err != nil {
			return fmt.Errorf("write reply: %w", err)
		}
		return nil
	}

	// guest
	c, err := net.DialTimeout("tcp", net.JoinHostPort(peerIP, gamePortTCP), 10*time.Second)
	if err != nil {
		return fmt.Errorf("dial peer: %w", err)
	}
	defer c.Close()
	c.SetWriteDeadline(deadline)
	// Send magic + payload as a single write (TCP may coalesce anyway).
	msg := append(append([]byte{}, magic...), payload...)
	if _, err := c.Write(msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	c.SetReadDeadline(deadline)
	buf := make([]byte, 1024)
	n, err := c.Read(buf)
	if err != nil {
		return fmt.Errorf("read reply: %w", err)
	}
	if string(buf[:n]) != "tcp-reply-from-host" {
		return fmt.Errorf("unexpected tcp reply: %q", string(buf[:n]))
	}
	return nil
}

// encodePacket builds a backend wire frame: [255][code][len:2 LE][payload].
func encodePacket(code byte, payload []byte) []byte {
	buf := make([]byte, 4+len(payload))
	buf[0] = 255
	buf[1] = code
	binary.LittleEndian.PutUint16(buf[2:4], uint16(len(buf)))
	copy(buf[4:], payload)
	return buf
}

func writeFrame(conn net.Conn, code byte, payload []byte) error {
	_, err := conn.Write(encodePacket(code, payload))
	return err
}

// readFrame reads one game packet: [255, code, 2-byte total-length, payload].
// The 2-byte length is the TOTAL frame size (including the 4-byte header).
func readFrame(conn net.Conn) ([]byte, error) {
	hdr := make([]byte, 4)
	if _, err := io.ReadFull(conn, hdr); err != nil {
		return nil, fmt.Errorf("read frame header: %w", err)
	}
	if hdr[0] != 255 {
		return nil, fmt.Errorf("unexpected frame marker 0x%02x", hdr[0])
	}
	total := int(binary.LittleEndian.Uint16(hdr[2:4]))
	if total < 4 || total > 1<<20 {
		return nil, fmt.Errorf("invalid frame length %d", total)
	}
	payload := make([]byte, total-4)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, fmt.Errorf("read frame payload: %w", err)
	}
	return payload, nil
}

func pad(b []byte, n int) []byte {
	if len(b) >= n {
		return b[:n]
	}
	out := make([]byte, n)
	copy(out, b)
	return out
}
