package relayserver

import (
	"context"
	"crypto/tls"
	_ "embed"
	"net"
	"time"

	"github.com/quic-go/quic-go"
)

//go:embed cert.pem
var devCertPEM []byte

//go:embed key.pem
var devKeyPEM []byte

// RelayListener is the transport seam for accepting relay connections. It
// decouples RelayServer from *quic.Listener so an in-memory implementation can
// be injected in tests (and the relay could later run over a different
// transport) without touching the relay logic.
type RelayListener interface {
	Accept(ctx context.Context) (RelayConn, error)
	Addr() net.Addr
	Close() error
}

// quicListenerAdapter adapts *quic.Listener to RelayListener.
type quicListenerAdapter struct {
	l *quic.Listener
}

func (a *quicListenerAdapter) Accept(ctx context.Context) (RelayConn, error) {
	conn, err := a.l.Accept(ctx)
	if err != nil {
		return nil, err
	}
	return &quicConnAdapter{conn: conn}, nil
}

func (a *quicListenerAdapter) Addr() net.Addr { return a.l.Addr() }

func (a *quicListenerAdapter) Close() error { return a.l.Close() }

// quicConnAdapter adapts quic.Connection to RelayConn. Its AcceptStream returns
// the underlying *quic.Stream as a RelayStream.
type quicConnAdapter struct {
	conn *quic.Conn
}

func (a *quicConnAdapter) AcceptStream(ctx context.Context) (RelayStream, error) {
	s, err := a.conn.AcceptStream(ctx)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (a *quicConnAdapter) CloseWithError(code quic.ApplicationErrorCode, msg string) error {
	return a.conn.CloseWithError(code, msg)
}

func (a *quicConnAdapter) RemoteAddr() net.Addr { return a.conn.RemoteAddr() }

// newQUICListener creates a real QUIC listener bound to addr using the
// self-signed "game-relay" ALPN configuration.
func newQUICListener(addr string) (RelayListener, error) {
	cert, err := generateSelfSigned()
	if err != nil {
		return nil, err
	}

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"game-relay"},
		Certificates:       []tls.Certificate{cert},
	}

	listener, err := quic.ListenAddr(addr, tlsConf, &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 15 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &quicListenerAdapter{l: listener}, nil
}

// generateSelfSigned loads the embedded development certificate.
func generateSelfSigned() (tls.Certificate, error) {
	return tls.X509KeyPair(devCertPEM, devKeyPEM)
}

// WithListener injects a RelayListener, overriding the default real QUIC
// listener created by NewRelayServer. Used by tests to run the relay server
// fully in-process.
func WithListener(l RelayListener) RelayServerOption {
	return func(rs *RelayServer) { rs.listener = l }
}