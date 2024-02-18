package proxy

// import (
// 	"context"
// 	"fmt"
// 	"net"
// )

// type EventType = int

// const (
// 	NoEvent EventType = iota
// 	ClosedConnection
// 	WrotePacket
// 	Other
// )

// type Event struct {
// 	Type EventType
// }

// // NewServerProxy is called when new server is created
// func NewServerProxy(ctx context.Context, events chan Event) error {
// 	gameServerIP := "127.0.0.1"

// 	// Establish connection to the game server
// 	tcpConn, err := net.Dial("tcp", fmt.Sprintf("%s:6114", gameServerIP))
// 	if err != nil {
// 		return err
// 	}
// 	defer tcpConn.Close()

// 	udpConn, err := net.Dial("udp", fmt.Sprintf("%s:6113", gameServerIP))
// 	if err != nil {
// 		return err
// 	}
// 	defer udpConn.Close()

// 	ListenEventPipe(ctx, events, tcpConn, udpConn)
// 	return nil
// }

// func ListenEventPipe(ctx context.Context, events chan Event, tcpConn net.Conn, udpConn net.Conn) error {
// 	for {
// 		select {
// 		case event := <-events:
// 			if events == nil {
// 				break
// 			}

// 			switch event.Type {
// 			case NoEvent:
// 				// Do nothing
// 				break
// 			}
// 		case <-ctx.Done():
// 			return ctx.Err()
// 		}
// 	}
// }
