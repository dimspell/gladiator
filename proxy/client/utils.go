package client

import (
	"log/slog"

	"github.com/fxamacker/cbor/v2"
)

func decodeCBOR[T any](data []byte) (v T, err error) {
	err = cbor.Unmarshal(data, &v)
	if err != nil {
		slog.Warn("Error decoding JSON", "error", err, "payload", string(data))
	}
	return
}
