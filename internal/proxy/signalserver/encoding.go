package signalserver

import (
	"encoding/json"

	"github.com/fxamacker/cbor/v2"
)

// var DefaultCodec = NewCBORCodec()
var DefaultCodec = NewJSONCodec()

type Codec struct {
	Marshal   func(v any) ([]byte, error)
	Unmarshal func(data []byte, v any) error
}

func NewJSONCodec() *Codec {
	return &Codec{
		Marshal:   json.Marshal,
		Unmarshal: json.Unmarshal,
	}
}

func NewCBORCodec() *Codec {
	return &Codec{
		Marshal:   cbor.Marshal,
		Unmarshal: cbor.Unmarshal,
	}
}
