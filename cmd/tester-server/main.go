package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"golang.org/x/sync/errgroup"
)

const backendIP = "127.0.1.28"

type Proxy struct {
	TCPHost string
	TCPPort string
	UDPHost string
	UDPPort string
}

func main() {
	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxy := Proxy{
		TCPHost: "127.0.0.1",
		TCPPort: "6114",
		UDPHost: "127.0.0.1",
		UDPPort: "6113",
	}

	// Start TCP listener
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return proxy.listenTCP(ctx)
	})
	g.Go(func() error {
		return proxy.listenUDP(ctx)
	})

	if err := g.Wait(); err != nil {
		cancel()
		slog.Error(err.Error())
		return
	}
}

func (p *Proxy) listenUDP(ctx context.Context) error {
	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(p.UDPHost, p.UDPPort))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to start UDP listener: %w", err)
	}
	defer udpConn.Close()

	slog.Info("Listening UDP", "addr", udpAddr.String())

	buf := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			udpConn.Close()
			return nil
		default:
			clear(buf)
			n, addr, err := udpConn.ReadFrom(buf)
			if err != nil {
				return fmt.Errorf("UDP read error: %w", err)
			}

			slog.Debug("UDP packet received", "data", buf[:n], "from", addr.String())

			if buf[0] == 26 {
				response := []byte{27, 0, 2, 0}
				if _, err = udpConn.WriteTo(response, addr); err != nil {
					slog.Error("Failed to send UDP response", logging.Error(err))
				}
				slog.Debug("Sent UDP response", "data", response)
			}
		}
	}

	// for {
	// 	if ctx.Err() != nil {
	// 		fmt.Println("context err")
	// 		return
	// 	}
	//
	// 	buf := make([]byte, 1024)
	// 	n, addr, err := udpConn.ReadFrom(buf)
	// 	if err != nil {
	// 		break
	// 	}
	//
	// 	fmt.Println("Accepted UDP connection", connPort, addr.String(), buf[:n])
	//
	// 	if buf[0] == 26 {
	// 		{
	// 			_, err = udpConn.WriteToUDP([]byte{27, 0, 2, 0}, udpAddr)
	// 			fmt.Println(err)
	// 		}
	// 		fmt.Println("Responded with 27")
	// 	}
	// }
}

func (p *Proxy) listenTCP(ctx context.Context) error {
	l, err := net.Listen("tcp", net.JoinHostPort(p.TCPHost, p.TCPPort))
	if err != nil {
		return fmt.Errorf("failed to start TCP listener: %w", err)
	}
	defer l.Close()

	slog.Info("Listening TCP", "addr", l.Addr().String())

	go func() {
		<-ctx.Done()
		l.Close()
	}()

	handleConnection := func(conn net.Conn) {
		defer conn.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				buf := make([]byte, 1024)
				n, err := conn.Read(buf)
				if err != nil {
					slog.Error("TCP: error reading", logging.Error(err))
					return
				}

				slog.Debug("TCP packet received", "data", buf[:n], "string", string(buf[:n]))

				if _, err := conn.Write([]byte{35, 35, 116, 101, 115, 116, 0}); err != nil {
					slog.Error("Failed to send TCP response", logging.Error(err))
				}
			}
		}
	}
 
	// Close the listener when the application closes.
	defer l.Close()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Listen for an incoming connection.
			conn, err := l.Accept()
			if err != nil {
				slog.Error("Error accepting TCP connection", logging.Error(err))
				return err
			}
			slog.Info("Accepted new TCP connection", "addr", conn.RemoteAddr().String())

			go handleConnection(conn)
		}
	}
}
