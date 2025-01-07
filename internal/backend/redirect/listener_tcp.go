package redirect

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"golang.org/x/sync/errgroup"
)

var _ Redirect = (*ListenerTCP)(nil)

type ListenerTCP struct {
	listenerTCP net.Listener
	logger      *slog.Logger

	mu      sync.RWMutex
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

	logger := slog.With(
		slog.String("redirect", "listen-tcp"),
		slog.String("addr", tcpListener.Addr().String()),
	)
	logger.Info("Listening TCP")

	return &ListenerTCP{
		listenerTCP: tcpListener,
		logger:      logger,
	}, nil
}

func (p *ListenerTCP) Run(ctx context.Context, rw io.Writer) error {
	defer func() {
		p.logger.Info("Closing listener")

		if err := p.listenerTCP.Close(); err != nil {
			p.logger.Error("Could not close listener", "error", err)
			return
		}
	}()

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		conn, err := p.listenerTCP.Accept()
		if err != nil {
			p.logger.Error("failed to accept the TCP connection", "error", err)
			continue
		}
		p.logger.Debug("Accepted connection", "remote-addr", conn.RemoteAddr())

		go p.readerTCP(ctx, conn, rw)
	}
}

func (p *ListenerTCP) readerTCP(ctx context.Context, conn net.Conn, dc io.Writer) {
	p.mu.Lock()
	defer func() {
		p.logger.Warn("Closing TCP connection from listener", "try-error", conn.Close())
		p.mu.Unlock()
	}()

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		for {
			if gCtx.Err() != nil {
				return gCtx.Err()
			}
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				p.logger.Warn("Error reading from TCP", "error", err)
				return err
			}
			if _, err := dc.Write(buf[:n]); err != nil {
				p.logger.Warn("Error writing to TCP", "error", err)
				return err
			}

			p.logger.Debug("Received from client", "payload", buf[0:n])
		}

	})
	g.Go(func() error {
		<-ctx.Done()
		conn.Close()
		return ctx.Err()
	})

	if err := g.Wait(); err != nil {
		p.logger.Error("Failed proxying TCP", "error", err)
	}
}

func (p *ListenerTCP) Write(msg []byte) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.connTCP == nil {
		return 0, fmt.Errorf("listener-tcp: no connection available")
	}
	return p.connTCP.Write(msg)
}

func (p *ListenerTCP) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errConn, errListener error

	// Close an existing TCP connection if any
	if p.connTCP != nil {
		errConn = p.connTCP.Close()
		p.connTCP = nil
	}

	// Close the TCP listener
	errListener = p.listenerTCP.Close()
	p.listenerTCP = nil

	return errors.Join(errConn, errListener)
}
