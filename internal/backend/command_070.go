package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/model"
)

// HandleShowRanking handles 0x46ff (255-70) command
func (b *Backend) HandleShowRanking(session *Session, req RankingRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-70: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", "error", err)
		return nil
	}

	respRanking, err := b.rankingClient.GetRanking(context.TODO(),
		connect.NewRequest(&multiv1.GetRankingRequest{
			UserId:        session.UserID,
			CharacterName: data.CharacterName,
			ClassType:     int64(data.ClassType),
			Offset:        int64(data.Offset),
		}))
	if err != nil {
		return err
	}

	ranking := model.RankingToBytes(respRanking.Msg)

	return session.Send(ShowRanking, ranking)
}

type RankingRequest []byte

type RankingRequestData struct {
	ClassType     model.ClassType
	Offset        uint32
	Username      string
	CharacterName string
}

func NewRankingRequest(data RankingRequestData) RankingRequest {
	userNameLength := len(data.Username)
	characterNameLength := len(data.CharacterName)

	buf := make([]byte, 4+4+userNameLength+1+characterNameLength+1)

	// 0:4 Class type
	buf[0] = byte(data.ClassType)

	// 4:8 Offset used in pagination
	binary.LittleEndian.PutUint32(buf[4:8], data.Offset)

	// Username (null-terminated string)
	copy(buf[8:], data.Username)
	buf[8+userNameLength] = 0

	// Character name (null-terminated string)
	copy(buf[8+userNameLength+1:], data.CharacterName)

	return buf
}

func (r RankingRequest) Parse() (data RankingRequestData, err error) {
	rd := packet.NewReader(r)

	classType, err := rd.ReadUint32()
	if err != nil {
		return data, fmt.Errorf("packet-70: malformed class type: %w", err)
	}
	data.ClassType = model.ClassType(classType)

	data.Offset, err = rd.ReadUint32()
	if err != nil {
		return data, fmt.Errorf("packet-70: malformed offset: %w", err)
	}

	data.Username, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-70: malformed username: %w", err)
	}
	data.CharacterName, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-70: malformed character name: %w", err)
	}

	return data, rd.Close()
}
