package redirect

import (
	"context"
	"io"
	"log/slog"
	"net"
	"sync"

	"golang.org/x/sync/errgroup"
)

var _ Redirect = (*ListenerTCP)(nil)

type ListenerTCP struct {
	listenerTCP net.Listener

	mu      sync.Mutex
	connTCP net.Conn
}

func ListenTCP(ipv4 string, portNumber string) (*ListenerTCP, error) {
	if portNumber == "" {
		portNumber = "6114"
	}
	tcpListener, err := net.Listen("tcp", net.JoinHostPort(ipv4, portNumber))
	if err != nil {
		return nil, err
	}
	slog.Info("Guest: Listening TCP", "addr", tcpListener.Addr().String())

	return &ListenerTCP{
		listenerTCP: tcpListener,
	}, nil
}

func (p *ListenerTCP) Run(ctx context.Context, rw io.Writer) error {
	defer p.listenerTCP.Close()
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		conn, err := p.listenerTCP.Accept()
		if err != nil {
			slog.Error("error accepting", "error", err)
			continue
		}
		slog.Debug("Accepted connection on port", "port", conn.RemoteAddr(), "protocol", "tcp")

		go p.readerTCP(ctx, conn, rw)
	}
}

func (p *ListenerTCP) readerTCP(ctx context.Context, conn net.Conn, dc io.Writer) {
	p.mu.Lock()
	defer func() {
		slog.Warn("Closing connection to client", "protocol", "tcp")
		_ = conn.Close()
		p.mu.Unlock()
	}()

	g, ctx := errgroup.WithContext(ctx)
	// g.Go(func() error {
	// 	if _, err := io.Copy(conn, dc); err != nil {
	// 		slog.Warn("Error copying from client to server", "error", err, "protocol", "tcp")
	// 		return err
	// 	}
	// 	return nil
	// })
	g.Go(func() error {
		_, err := io.Copy(dc, conn)
		if err != nil {
			slog.Warn("Error copying from server to client", "error", err, "protocol", "tcp")
		}
		return err
	})
	g.Go(func() error {
		select {
		case <-ctx.Done():
			return conn.Close()
		}
	})

	// g.Go(func() error {
	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 			return ctx.Err()
	// 		case msg, ok := <-p.writeTCP:
	// 			if !ok {
	// 				return fmt.Errorf("closed channel")
	// 			}
	// 			if _, err := conn.Write(msg); err != nil {
	// 				slog.Warn("Error writing to TCP", "error", err, "protocol", "tcp")
	// 				return err
	// 			}
	// 		}
	// 	}
	// })

	if err := g.Wait(); err != nil {
		slog.Error("Failed proxying TCP", "error", err)
	}
}

func (p *ListenerTCP) Write(msg []byte) (int, error) {
	if p.connTCP == nil {
		return 0, io.EOF
	}
	return p.connTCP.Write(msg)
}

func (p *ListenerTCP) Close() error {
	return p.connTCP.Close()
}
