package probe

import (
	"net"
	"time"
)

type TCPChecker struct {
	Timeout time.Duration
	Address string
}

func (c *TCPChecker) Check() error {
	conn, err := net.DialTimeout("tcp", c.Address, c.Timeout)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
