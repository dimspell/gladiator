package turn

import (
	"fmt"
	"log"
	"net"
	"regexp"

	"github.com/pion/stun/v2"
	"github.com/pion/turn/v3"
)

type Config struct {
	// IP Address that TURN can be contacted by
	PublicIPAddr string

	// Listening port
	PortNumber string

	// TURN realm
	Realm string

	// List of username and password (e.g. "user=pass,user=pass")
	Users string
}

func Start() *turn.Server {
	cfg := &Config{
		Realm:        "dispelmulti.net",
		Users:        `username1=password1,username2=password2`,
		PublicIPAddr: "127.0.0.1",
		PortNumber:   "3478",
	}
	turnServer, err := StartTURNServer(cfg)
	if err != nil {
		log.Panicf("Could not start TURN server: %s", err)
	}
	return turnServer
}

func StartTURNServer(cfg *Config) (*turn.Server, error) {
	// Create a UDP listener to pass into pion/turn
	// pion/turn itself doesn't allocate any UDP sockets, but lets the user pass them in
	// this allows us to add logging, storage or modify inbound/outbound traffic
	udpListener, err := net.ListenPacket("udp4", net.JoinHostPort("0.0.0.0", cfg.PortNumber))
	if err != nil {
		log.Panicf("Failed to create TURN server listener: %s", err)
	}

	// Cache -users flag for easy lookup later
	// If passwords are stored they should be saved to your DB hashed using turn.GenerateAuthKey
	usersMap := map[string][]byte{}
	for _, kv := range regexp.MustCompile(`(\w+)=(\w+)`).FindAllStringSubmatch(cfg.Users, -1) {
		usersMap[kv[1]] = turn.GenerateAuthKey(kv[1], cfg.Realm, kv[2])
	}

	s, err := turn.NewServer(turn.ServerConfig{
		Realm: cfg.Realm,
		// Set AuthHandler callback
		// This is called every time a user tries to authenticate with the TURN server
		// Return the key for that user, or false when no user is found
		AuthHandler: func(username string, realm string, srcAddr net.Addr) ([]byte, bool) { // nolint: revive
			if key, ok := usersMap[username]; ok {
				return key, true
			}
			return nil, false
		},
		// PacketConnConfigs is a list of UDP Listeners and the configuration around them
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: &stunLogger{udpListener},
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(cfg.PublicIPAddr), // Claim that we are listening on IP passed by user (This should be your Public IP)
					Address:      "0.0.0.0",                     // But actually be listening on every interface
				},
			},
		},
	})

	return s, err
}

// stunLogger wraps a PacketConn and prints incoming/outgoing STUN packets
// This pattern could be used to capture/inspect/modify data as well
type stunLogger struct {
	net.PacketConn
}

func (s *stunLogger) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if n, err = s.PacketConn.WriteTo(p, addr); err == nil && stun.IsMessage(p) {
		msg := &stun.Message{Raw: p}
		if err = msg.Decode(); err != nil {
			return
		}

		fmt.Printf("Outbound STUN: %s, %s \n", msg.String(), msg.Type)
	}

	return
}

func (s *stunLogger) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if n, addr, err = s.PacketConn.ReadFrom(p); err == nil && stun.IsMessage(p) {
		msg := &stun.Message{Raw: p}
		if err = msg.Decode(); err != nil {
			return
		}

		fmt.Printf("Inbound STUN: %s \n", msg.String())
	}

	return
}
