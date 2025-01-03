package command

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
