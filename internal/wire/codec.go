package wire

import (
	"encoding/json"
	"io"
	"log/slog"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
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

	payload := MustEncode(msg)
	payload = append([]byte{byte(msgType)}, payload...)
	return payload
}

func ComposeTyped[T any](msgType EventType, msg MessageContent[T]) []byte {
	msg.Type = msgType

	payload := MustEncode(msg)
	payload = append([]byte{byte(msgType)}, payload...)
	return payload
}

func MustEncode(m any) []byte {
	out, err := Encode(m)
	if err != nil {
		panic(err)
	}
	return out
}

func Encode(m any) ([]byte, error) {
	out, err := DefaultCodec.Marshal(m)
	if err != nil {
		slog.Error("Could not marshal the websocket message", logging.Error(err))
		return nil, err
	}
	return out, nil
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

// func DecodeAndRun[T any, S any](data []byte, f func(T, S) error, s S) error {
// 	_, v, err := DecodeTyped[T](data)
// 	if err != nil {
// 		return err
// 	}
// 	return f(v.Content, s)
// }

func ParseEventType(payload []byte) EventType {
	if len(payload) == 0 {
		return 0
	}
	return EventType(payload[0])
}
