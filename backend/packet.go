package backend

import (
	"bytes"
	"encoding/binary"

	"github.com/dispel-re/dispel-multi/model"
)

type PacketType byte

const (
	AuthorizationHandshake   PacketType = 6   // 0x6ff
	ListGames                PacketType = 9   // 0x9ff
	ListChannels             PacketType = 11  // 0xbff
	SelectedChannel          PacketType = 12  // 0xcff
	SendLobbyMessage         PacketType = 14  // 0xeff
	ReceiveMessage           PacketType = 15  // 0xfff
	PingClockTime            PacketType = 21  // 0x15ff
	CreateGame               PacketType = 28  // 0x1cff
	ClientHostAndUsername    PacketType = 30  // 0x1eff
	JoinGame                 PacketType = 34  // 0x22ff
	ClientAuthentication     PacketType = 41  // 0x29ff
	CreateNewAccount         PacketType = 42  // 0x2aff
	UpdateCharacterInventory PacketType = 44  // 0x2cff
	GetCharacters            PacketType = 60  // 0x3cff
	DeleteCharacter          PacketType = 61  // 0x3dff
	GetCharacterInventory    PacketType = 68  // 0x44ff
	SelectGame               PacketType = 69  // 0x45ff
	ShowRanking              PacketType = 70  // 0x46ff
	ChangeHost               PacketType = 71  // 0x47ff
	GetCharacterSpells       PacketType = 72  // 0x48ff
	UpdateCharacterSpells    PacketType = 73  // 0x49ff
	SelectCharacter          PacketType = 76  // 0x4cff
	CreateCharacter          PacketType = 92  // 0x5cff
	UpdateCharacterStats     PacketType = 108 // 0x6cff
)

type HandleAuthorizationHandshakeRequest []byte

type ClientHostAndUsernameRequest []byte

type ClientAuthenticationRequest []byte

func (r ClientAuthenticationRequest) Unknown() uint32 {
	return binary.LittleEndian.Uint32(r[0:4])
}

func (r ClientAuthenticationRequest) UsernameAndPassword() (username string, password string) {
	split := bytes.SplitN(r[4:], []byte{0}, 3)
	password = string(split[0])
	username = string(split[1])
	return username, password
}

type CreateNewAccountRequest []byte

func (r CreateNewAccountRequest) CDKey() uint32 {
	return binary.LittleEndian.Uint32(r[0:4])
}

func (r CreateNewAccountRequest) UsernameAndPassword() (username string, password string) {
	split := bytes.SplitN(r[4:], []byte{0}, 3)
	password = string(split[0])
	username = string(split[1])
	return username, password
}

func (r CreateNewAccountRequest) Unknown() []byte {
	split := bytes.SplitN(r[4:], []byte{0}, 3)
	return split[2]
}

type GetCharactersRequest []byte

func (r GetCharactersRequest) Username() string {
	return string(r[:len(r)-1])
}

type DeleteCharacterRequest []byte

func (r DeleteCharacterRequest) UsernameAndCharacterName() (username string, characterName string) {
	if bytes.Count(r, []byte{0}) < 2 {
		return "", ""
	}

	split := bytes.SplitN(r, []byte{0}, 3)
	username = string(split[0])
	characterName = string(split[1])
	return username, characterName
}

type CreateCharacterRequest []byte

func (r CreateCharacterRequest) CharacterInfo() model.CharacterInfo {
	return model.NewCharacterInfo(r[:56])
}

func (r CreateCharacterRequest) UserAndCharacterName() (username string, characterName string) {
	split := bytes.SplitN(r, []byte{0}, 3)
	username = string(split[0])
	characterName = string(split[1])
	return username, characterName
}

type SelectCharacterRequest []byte

func (r SelectCharacterRequest) UserAndCharacterName() (username string, characterName string) {
	split := bytes.SplitN(r, []byte{0}, 3)
	username = string(split[0])
	characterName = string(split[1])
	return username, characterName
}

type UpdateCharacterStatsRequest []byte

type GetCharacterInventoryRequest []byte

type UpdateCharacterInventoryRequest []byte

type GetCharacterSpellsRequest []byte

type UpdateCharacterSpellsRequest []byte

type ListChannelsRequest []byte

type SelectChannelRequest []byte

func (r SelectChannelRequest) ChannelName() string {
	split := bytes.Split(r, []byte{0})
	return string(split[0])
}

type SendLobbyMessageRequest []byte

func (c SendLobbyMessageRequest) Message() []byte {
	return c[:]
}

type CreateRoomRequest []byte

func (c CreateRoomRequest) State() uint32 {
	return binary.LittleEndian.Uint32(c[0:4])
}

func (c CreateRoomRequest) MapID() uint32 {
	return binary.LittleEndian.Uint32(c[4:8])
}

func (c CreateRoomRequest) NameAndPassword() (roomName string, password string) {
	split := bytes.Split(c[8:], []byte{0})
	roomName = string(split[0])
	password = string(split[1])
	return roomName, password
}

type ListGamesRequest []byte

type SelectGameRequest []byte

type JoinGameRequest []byte

type PingRequest []byte

func (r PingRequest) Milliseconds() uint32 {
	return binary.LittleEndian.Uint32(r[0:4])
}

type RankingRequest []byte

func NewRankingRequest(classType model.ClassType, offset uint32, userName string, characterName string) RankingRequest {
	userNameLength := len(userName)
	characterNameLength := len(characterName)

	buf := make([]byte, 4+4+userNameLength+1+characterNameLength+1)

	// 0:4 Class type
	buf[0] = byte(classType)

	// 4:8 Offset
	binary.LittleEndian.PutUint32(buf[4:8], offset)

	// Username
	copy(buf[8:], userName)
	buf[8+userNameLength] = 0

	// Character name
	copy(buf[8+userNameLength+1:], characterName)
	// buf[8+userNameLength+1+characterNameLength] = 0

	return buf
}

func (r RankingRequest) ClassType() model.ClassType {
	return model.ClassType(r[0])
}

func (r RankingRequest) Offset() uint32 {
	return binary.LittleEndian.Uint32(r[4:8])
}

func (r RankingRequest) UserAndCharacterName() (user string, character string) {
	split := bytes.Split(r[8:], []byte{0})
	user = string(split[0])
	character = string(split[1])
	return user, character
}
