package p2p

import "io"

type WebSocket interface {
	io.Closer
	io.Reader
	io.Writer
}
