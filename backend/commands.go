package backend

import (
	"encoding/binary"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleAuthorizationHandshake handles 0x6ff (255-6) command
func (b *Backend) HandleAuthorizationHandshake(session *model.Session, req HandleAuthorizationHandshakeRequest) error {
	response := append([]byte("ENET"), 0)
	return b.Send(session.Conn, AuthorizationHandshake, response)
}

// HandleClientHostAndUsername handles 0x1eff (255-30) command
func (b *Backend) HandleClientHostAndUsername(session *model.Session, req ClientHostAndUsernameRequest) error {
	return b.Send(session.Conn, ClientHostAndUsername, []byte{1, 0})
}

func (b *Backend) HandleClientAuthentication(session *model.Session, req ClientAuthenticationRequest) error {
	resp := make([]byte, 4)
	ok := true
	if ok {
		resp[0] = 1
	}
	return b.Send(session.Conn, ClientAuthentication, resp)
}

func (b *Backend) HandleCreateNewAccount(session *model.Session, req CreateNewAccountRequest) error {
	resp := make([]byte, 4)
	ok := true
	if ok {
		resp[0] = 1
	}
	return b.Send(session.Conn, CreateNewAccount, resp)
}

func (b *Backend) HandleGetCharacters(session *model.Session, req GetCharactersRequest) error {
	if len(session.User.Characters) == 0 {
		return b.Send(session.Conn, GetCharacters, []byte{0, 0, 0, 0})
	}

	response := []byte{1, 0, 0, 0}
	response = binary.LittleEndian.AppendUint32(response, uint32(len(session.User.Characters)))
	for _, character := range session.User.Characters {
		response = append(response, character.CharacterName...)
		response = append(response, 0)
	}
	return b.Send(session.Conn, GetCharacters, response)
}

func (b *Backend) HandleDeleteCharacter(session *model.Session, req DeleteCharacterRequest) error {
	_, characterName := req.UsernameAndCharacterName()

	var characters []model.Character
	for _, ch := range session.User.Characters {
		if ch.CharacterName != characterName {
			characters = append(characters, ch)
		}
	}
	session.User.Characters = characters

	response := make([]byte, len(characterName)+1)
	copy(response, characterName)

	return b.Send(session.Conn, DeleteCharacter, response)
}

func (b *Backend) HandleCreateCharacter(session *model.Session, req CreateCharacterRequest) error {
	info := req.CharacterInfo()
	_, character := req.UserAndCharacterName()

	newCharacter := model.Character{
		CharacterName: character,
		Slot:          0,
		Info:          info,
		Inventory:     model.CharacterInventory{},
		Spells:        nil,
	}
	session.User.Characters = append(session.User.Characters, newCharacter)

	return b.Send(session.Conn, CreateCharacter, []byte{1, 0, 0, 0})
}

func (b *Backend) HandleSelectCharacter(session *model.Session, req SelectCharacterRequest) error {
	_, characterName := req.UserAndCharacterName()

	for _, c := range session.User.Characters {
		if c.CharacterName == characterName {
			session.Character = &c
			break
		}
	}

	// No characters owned by player
	if session.Character == nil || session.Character.CharacterName != characterName {
		session.Character = nil
		return b.Send(session.Conn, SelectCharacter, []byte{0, 0, 0, 0})
	}

	// Provide stats of the selected character
	info := session.Character.Info
	response := make([]byte, 60)
	response[0] = 1
	copy(response[4:], info.ToBytes())

	return b.Send(session.Conn, SelectCharacter, response)
}

func (b *Backend) HandleUpdateCharacterStats(session *model.Session, req UpdateCharacterStatsRequest) error {
	// _ = CharacterFromBytes(buf)
	return b.Send(session.Conn, SelectCharacter, []byte{})
}

func (b *Backend) HandleGetCharacterInventory(session *model.Session, req GetCharacterInventoryRequest) error {
	resp := make([]byte, 207)

	for i, item := range session.Character.Inventory.Backpack {
		resp[0+i*3] = item.TypeId
		resp[1+i*3] = item.ItemId
		resp[2+i*3] = item.Unknown
	}
	for i, item := range session.Character.Inventory.Belt {
		resp[0+63+i*3] = item.TypeId
		resp[1+63+i*3] = item.ItemId
		resp[2+63+i*3] = item.Unknown
	}
	resp = append(resp, 0)

	return b.Send(session.Conn, GetCharacterInventory, resp)
}

func (b *Backend) HandleUpdateCharacterInventory(session *model.Session, req UpdateCharacterInventoryRequest) error {
	// rd := bufio.NewReader(bytes.NewReader(buf[4:]))
	// _, _ = rd.ReadBytes(0)         // username
	// _, _ = rd.ReadBytes(0)         // character
	// backpack, _ := rd.ReadBytes(0) // inventory
	// printBackpack(backpack)

	return b.Send(session.Conn, UpdateCharacterInventory, []byte{1, 0, 0, 0})
}

func (b *Backend) HandleGetCharacterSpells(session *model.Session, req GetCharacterSpellsRequest) error {
	// spells := make([]byte, 41)
	// for i := 0; i < len(spells); i++ {
	// 	spells[i] = 2
	// }
	// resp := []byte{255, opJoinedUpdateSpells, 0, 0}
	// resp = append(resp, spells...)
	// resp = append(resp, 0, 0)
	// binary.LittleEndian.PutUint16(resp[2:4], uint16(len(resp)))
	//
	// _, _ = conn.Write(resp)
	// _, err := conn.Write(resp)

	return nil
}

func (b *Backend) HandleUpdateCharacterSpells(session *model.Session, req UpdateCharacterSpellsRequest) error {
	// // <= [255 73 59 0 115 97 100 97 0 107 110 105 103 104 116 0 2 1 1 1 1 1 1 1 1 1 1 1 1 1 1 2 1 1 1 2 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 1 0 0]
	// rd := bufio.NewReader(bytes.NewReader(buf[4:]))
	// _, _ = rd.ReadBytes(0)       // username
	// _, _ = rd.ReadBytes(0)       // character
	// spells, _ := rd.ReadBytes(0) // spells
	// something, _ := rd.ReadBytes(0)
	// printSpells(spells)
	// fmt.Println(something)

	return nil
}

// HandleListChannels handles 0xbff (255-11) command
func (b *Backend) HandleListChannels(session *model.Session, req ListChannelsRequest) error {
	var response []byte
	channels, _ := b.DB.ListChannels()
	for _, channel := range channels {
		response = append(response, channel...)
		response = append(response, 0)
	}
	return b.Send(session.Conn, ListChannels, response)
}

func (b *Backend) HandleSelectChannel(session *model.Session, req SelectChannelRequest) error {
	if req.ChannelName() == "DISPEL" {
		// b.Send(session.Conn, ReceiveMessage, NewMessage())
	}

	return nil
}

func (b *Backend) HandleSendLobbyMessage(session *model.Session, req SendLobbyMessageRequest) error {
	resp := NewLobbyMessage(session.Character.CharacterName, string(req))
	return b.Send(session.Conn, ReceiveMessage, resp)
}

// HandleCreateGame handles 0x1cff (255-28) command
func (b *Backend) HandleCreateGame(session *model.Session, req CreateRoomRequest) error {
	resp := make([]byte, 4)

	switch req.State() {
	case uint32(0):
		binary.LittleEndian.PutUint32(resp[0:4], 1)
		// b.CreateGameRoom()
		break
	case uint32(1):
		binary.LittleEndian.PutUint32(resp[0:4], 2)
		break
	}

	return b.Send(session.Conn, CreateGame, resp)
}

// HandleListGames handles 0x9ff (255-9) command
func (b *Backend) HandleListGames(session *model.Session, req ListGamesRequest) error {
	response := []byte{}
	for _, room := range b.DB.GameRooms() {
		response = append(response, room.Lobby.ToBytes()...)
	}
	return b.Send(session.Conn, ListGames, response)
}

// HandleSelectGame handles 0x45ff (255-69) command
func (b *Backend) HandleSelectGame(session *model.Session, req SelectGameRequest) error {
	gameRoom := b.DB.GameRooms()[0]
	return b.Send(session.Conn, SelectGame, gameRoom.Details())
}

// HandleJoinGame handles 0x22ff (255-34) command
func (b *Backend) HandleJoinGame(session *model.Session, req JoinGameRequest) error {
	gameRoom := b.DB.GameRooms()[0]
	return b.Send(session.Conn, JoinGame, gameRoom.Details())
}

// HandleShowRanking handles 0x46ff (255-70) command
func (b *Backend) HandleShowRanking(session *model.Session, req RankingRequest) error {
	ranking := b.DB.Ranking()
	return b.Send(session.Conn, ShowRanking, ranking.ToBytes())
}

func (b *Backend) HandlePing(session *model.Session, req PingRequest) error {
	return b.Send(session.Conn, PingClockTime, []byte{1, 0, 0, 0})
}
