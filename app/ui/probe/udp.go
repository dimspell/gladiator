package probe

import (
	"net"
	"time"
)

// UDPChecker checks a UDP server by sending a UDP packet.
// Example:
//
//	udpChecker := &probe.UDPChecker{
//		Timeout: 5 * time.Second,
//		Address: "example.com:53",
//		Data:    []byte("ping"),
//	}
type UDPChecker struct {
	Timeout time.Duration
	Address string
	Data    []byte
}

func (c *UDPChecker) Check() error {
	conn, err := net.DialTimeout("udp", c.Address, c.Timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(c.Data)
	if err != nil {
		return err
	}

	return nil
}
