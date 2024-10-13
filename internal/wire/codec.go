package wire

import (
	"encoding/json"
	"io"

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

func Compose(msgType EventType, msg Message) []byte {
	msg.Type = msgType

	// TODO: Use msg.Encode()
	payload := msg.Encode()
	// if err != nil {
	// 	slog.Error("Could not marshal the websocket message", "error", err)
	// 	return nil
	// }
	payload = append([]byte{byte(msgType)}, payload...)
	return payload
}

func Decode(payload []byte) (et EventType, m Message, err error) {
	if len(payload) == 0 {
		return et, m, io.ErrShortBuffer
	}
	err = DefaultCodec.Unmarshal(payload[1:], &m)
	if err != nil {
		return et, m, err
	}
	return EventType(payload[0]), m, nil
}

func DecodeTyped[T any](payload []byte) (et EventType, m MessageContent[T], err error) {
	if len(payload) == 0 {
		return et, m, io.ErrShortBuffer
	}
	err = DefaultCodec.Unmarshal(payload[1:], &m)
	if err != nil {
		return et, m, err
	}
	return EventType(payload[0]), m, nil
}

func ParseEventType(payload []byte) EventType {
	if len(payload) == 0 {
		return 0
	}
	return EventType(payload[0])
}
