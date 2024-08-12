package p2p

import (
	"context"
	"io"
	"log"
	"log/slog"
	"net"

	"golang.org/x/sync/errgroup"
)

type ListenerTCP struct {
	connTCP  net.Listener
	writeTCP chan []byte
}

func ListenTCP(tcpAddr string) (*ListenerTCP, error) {
	tcpListener, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		return nil, err
	}
	slog.Info("Guest: Listening TCP", "addr", tcpListener.Addr().String())

	return &ListenerTCP{
		connTCP:  tcpListener,
		writeTCP: make(chan []byte),
	}, nil
}

func (p *ListenerTCP) Run(ctx context.Context, rw io.ReadWriteCloser) error {
	defer p.connTCP.Close()
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		conn, err := p.connTCP.Accept()
		if err != nil {
			log.Println("Error accepting: ", err.Error())
			continue
		}
		slog.Debug("Accepted connection on port", "port", conn.RemoteAddr(), "protocol", "tcp")
		go p.readerTCP(ctx, conn, rw)
	}
}

func (p *ListenerTCP) readerTCP(ctx context.Context, conn net.Conn, rw io.ReadWriteCloser) {
	defer func() {
		slog.Warn("Closing connection to client", "protocol", "tcp")
		conn.Close()
		rw.Close()
	}()

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if _, err := io.Copy(conn, rw); err != nil {
			slog.Warn("Error copying from client to server", "error", err, "protocol", "tcp")
			return err
		}
		return nil
	})
	g.Go(func() error {
		_, err := io.Copy(rw, conn)
		if err != nil {
			slog.Warn("Error copying from server to client", "error", err, "protocol", "tcp")
		}
		return err
	})
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case msg := <-p.writeTCP:
				if _, err := conn.Write(msg); err != nil {
					slog.Warn("Error writing to TCP", "error", err, "protocol", "tcp")
					return err
				}
			}
		}
	})

	if err := g.Wait(); err != nil {
		slog.Error("Failed proxying TCP", "error", err)
	}
}

func (p *ListenerTCP) Write(msg []byte) (int, error) {
	if p.connTCP == nil {
		return 0, io.EOF
	}
	p.writeTCP <- msg
	return len(msg), nil
}

func (p *ListenerTCP) Close() error {
	return p.connTCP.Close()
}
