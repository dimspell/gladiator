package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"github.com/fxamacker/cbor/v2"
	"golang.org/x/sync/errgroup"
)

var isUnitTest bool

func writeCBOR[T any](conn io.Writer, command string, value T) error {
	// data, err := cbor.Marshal(value)
	// if err != nil {
	// 	return err
	// }
	//
	// prefix := make([]byte, 5)
	// copy(prefix, command)

	// if _, err := conn.Write(slices.Concat(prefix, data)); err != nil {
	// 	return err
	// }
	return nil
}

func parseCBOR[T any](data []byte) (value T, err error) {
	if err := cbor.Unmarshal(data, &value); err != nil {
		return value, err
	}
	return value, nil
}

type Wire struct {
	isConnected bool

	Address         string
	GlobalProxyConn net.Conn

	tcpDataCh chan Data
	udpDataCh chan Data
	closeCh   chan struct{}

	Me      *Player
	Host    *Player
	Players map[int]*Player
}

func NewWire(addr string) *Wire {
	return &Wire{
		isConnected: false,

		tcpDataCh: make(chan Data),
		udpDataCh: make(chan Data),
		closeCh:   make(chan struct{}),
	}
}

type Credentials struct {
	//
}

type Player struct {
	IP       net.IP
	PlayerID string
}

func (p *Wire) Start(credentials *Credentials, host *Player, me *Player) error {
	if p.isConnected {
		return nil
	}
	p.Host = host
	p.Me = me

	if err := p.connect(credentials); err != nil {
		return err
	}

	go func() {
		p.startTCP(context.TODO())
	}()
	return nil
}

func (p *Wire) UDP() {

}

func (p *Wire) Stop() {
	if p == nil {
		return
	}
	// Close all channels

	if p.closeCh != nil {
		close(p.closeCh)
	}

	// p.GlobalProxyConn = nil
	// clear(p.MapIPIndexToUser)
	// clear(p.MapIPIndexToUser)
}

const (
	// CommandHello is used for handshake & authorization to public proxy
	CommandHello = "HELLO"

	// CommandDATA is used to receive data from the server
	CommandDATA = "DATA"
)

type HelloInput struct {
	Username string
	AuthKey  string
}

type HelloOutput struct {
	GameKey string
}

type Data struct {
	GameKey      string
	FromPlayerID string
	ToPlayerID   string
	Protocol     string
	Payload      []byte
}

func (p *Wire) connect(credentials *Credentials) error {
	if !isUnitTest {
		conn, err := net.DialTimeout("tcp", p.Address, DefaultConnectionTimeout)
		if err != nil {
			return err
		}
		go func() {
			<-p.closeCh
			if err := conn.Close(); err != nil {
				slog.Warn("dev-proxy: received close signal - closing connection")
			}
		}()

		p.GlobalProxyConn = conn
	}

	// Handshake and authorize
	hello, err := p.hello(p.GlobalProxyConn)
	if err != nil {
		return fmt.Errorf("proxy: could not authorize with hello command: %w", err)
	}
	slog.Info("Received HELLO", "gameKey", hello.GameKey)

	// Broker of packets received from the global proxy
	go p.parsePackets(p.GlobalProxyConn)

	return nil
}

func (p *Wire) parsePackets(conn net.Conn) {
	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil {
		p.closeCh <- struct{}{}
		return
	}
	if n < 5 {
		p.closeCh <- struct{}{}
		return
	}

	switch true {
	case bytes.HasPrefix(buf[:5], []byte("DATAT")):
		output, err := parseCBOR[Data](buf[5:n])
		if err != nil {
			break
		}
		p.tcpDataCh <- output
		break
	case bytes.HasPrefix(buf[:5], []byte("DATAU")):
		output, err := parseCBOR[Data](buf[5:n])
		if err != nil {
			break
		}
		p.udpDataCh <- output
		break
	case bytes.HasPrefix(buf[:5], []byte("JOIN")):
		break
	case bytes.HasPrefix(buf[:5], []byte("CLOSE")):
		break
	}
}

func (p *Wire) hello(conn net.Conn) (*HelloOutput, error) {
	if err := writeCBOR(p.GlobalProxyConn,
		"HELLO",
		HelloInput{Username: p.Me.PlayerID},
	); err != nil {
		return nil, err
	}

	buf := make([]byte, 128)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("could not read packet: %w", err)
	}
	if n < 5 {
		return nil, fmt.Errorf("received invalid length of hello from the proxy server")
	}
	if !bytes.Equal(buf[:5], []byte("HELLO")) {
		return nil, fmt.Errorf("expected HELLO command to be returned from the proxy server")
	}

	output, err := parseCBOR[HelloOutput](buf[5:n])
	if err != nil {
		return nil, fmt.Errorf("could not parse HELLO command: %w", err)
	}
	return &output, nil
}

func (p *Wire) startUDP(ctx context.Context, index int, ip net.IP) error {
	slog.Info("Starting proxy for UDP", "ip", ip.String())

	srcAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ip.To4().String(), "6113"))
	if err != nil {
		fmt.Println("Error resolving UDP address: ", err.Error())
		return err
	}
	srcConn, err := net.ListenUDP("udp", srcAddr)
	if err != nil {
		fmt.Println("Error listening on UDP: ", err.Error())
		return err
	}
	defer srcConn.Close()

	var (
		clientDestAddr *net.UDPAddr
		clientDestOnce sync.Once
	)
	setClientAddr := func(addr *net.UDPAddr) func() {
		return func() { clientDestAddr = addr }
	}

	group, ctx := errgroup.WithContext(ctx)

	// Host -> Game
	group.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case data := <-p.udpDataCh:
				if data.Payload == nil {
					return fmt.Errorf("empty payload")
				}
				if clientDestAddr == nil {
					continue
				}

				if _, err := srcConn.WriteToUDP(data.Payload, clientDestAddr); err != nil {
					return err
				}
				fmt.Println("(udp): (server): wrote to client", data.Payload)
			}
		}
	})

	// Game -> Host
	group.Go(func() error {
		<-p.closeCh
		return srcConn.Close()
	})
	group.Go(func() error {
		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			buf := make([]byte, 1024)
			n, addr, err := srcConn.ReadFromUDP(buf)
			if err != nil {
				fmt.Println("(udp): Error reading from client: ", err)
				return err
			}
			clientDestOnce.Do(setClientAddr(addr))

			fmt.Println("(udp): (client): Received ", (buf[0:n]), " from ", addr)

			// to, ok := p.MapIPIndexToUser[index]
			// if !ok {
			// 	return fmt.Errorf("Unknown ip address")
			// }

			err = writeCBOR[Data](p.GlobalProxyConn, "DATAU", Data{Payload: buf[0:n]})
			if err != nil {
				fmt.Println("(udp): Error writing to server: ", err)
				return err
			}
			fmt.Println("(udp): (client): wrote to server", buf[0:n])
		}
	})

	if err := group.Wait(); err != nil {
		p.closeCh <- struct{}{}
		return err
	}

	return nil
}

func (p *Wire) startTCP(ctx context.Context) error {
	tcpFakeHost, err := net.Listen("tcp", net.JoinHostPort(p.Host.IP.To4().String(), "6114"))
	if err != nil {
		fmt.Println("Error listening on TCP:", err.Error())
		return err
	}
	defer tcpFakeHost.Close()
	fmt.Println("Listening TCP on", tcpFakeHost.Addr().String())

	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error {
		<-p.closeCh
		return fmt.Errorf("closed connection")
	})
	group.Go(func() error {
		processPackets := func(connGameClient net.Conn) {
			defer connGameClient.Close()

			ctx, cancel := context.WithCancel(context.TODO())
			defer cancel()

			// Master Proxy -> Fake Host
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case msg := <-p.tcpDataCh:
						if msg.Payload == nil {
							return
						}
						_, err = connGameClient.Write(msg.Payload)
						if err != nil {
							fmt.Println("(tcp): Error writing to client: ", err.Error())
						}
					}
				}
			}()

			// Fake Host -> Master Proxy
			for {
				if ctx.Err() != nil {
					return
				}

				buf := make([]byte, 1024)
				n, err := connGameClient.Read(buf)
				if err != nil {
					fmt.Println("(tcp): Error reading from client: ", err.Error())
					return
				}

				if err := writeCBOR[Data](p.GlobalProxyConn, "DATAT", Data{
					Payload:      buf[:n],
					FromPlayerID: p.Me.PlayerID,
					ToPlayerID:   "host", // TODO: Name the host
				}); err != nil {
					return
				}
			}
		}

		for {
			gameConn, err := tcpFakeHost.Accept()
			if err != nil {
				fmt.Println("Error accepting: ", err.Error())
				continue
			}
			fmt.Println("(tcp): Accepted connection on port", gameConn.RemoteAddr())

			// TODO: Use workgroup
			go processPackets(gameConn)
		}
	})

	return group.Wait()
}
