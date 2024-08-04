package client

import (
	"container/ring"
	"log/slog"
	"net"

	"github.com/fxamacker/cbor/v2"
)

func decodeCBOR[T any](data []byte) (v T, err error) {
	err = cbor.Unmarshal(data, &v)
	if err != nil {
		slog.Warn("Error decoding JSON", "error", err, "payload", string(data))
	}
	return
}

type IpRing struct {
	*ring.Ring
}

func NewIpRing() IpRing {
	r := ring.New(100)
	n := r.Len()
	for i := 0; i < n; i++ {
		r.Value = i + 2
		r = r.Next()
	}
	return IpRing{r}
}

func (r *IpRing) IP() net.IP {
	d := byte(r.Value.(int))
	defer r.Next()
	return net.IPv4(127, 0, 1, d)
}
