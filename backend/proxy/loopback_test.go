package proxy

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDevProxy_Create(t *testing.T) {
	p, err := NewLoopback("127.0.1.1")
	if err != nil {
		assert.Error(t, err)
	}
	defer p.Close()
	p.Test = true

	conn := &fakeConn{
		BufWrite: bytes.NewBuffer([]byte{}),
		BufRead:  bytes.NewBuffer([]byte{}),
	}
	_ = writeCBOR(conn.BufRead, "HELLO", HelloOutput{"TESTER"})
	conn.BufRead.Write([]byte{'|'})

	p.GlobalProxyConn = conn

	ip, err := p.Create("", "archer")
	fmt.Println(ip, err)

	fmt.Println(conn.BufRead.String())
	fmt.Println(p)
}
