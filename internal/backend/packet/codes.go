package packet

type Code byte

const (
	AuthorizationHandshake   Code = 6   // 0x6ff
	ListGames                Code = 9   // 0x9ff
	ListChannels             Code = 11  // 0xbff
	SelectedChannel          Code = 12  // 0xcff
	SendLobbyMessage         Code = 14  // 0xeff
	ReceiveMessage           Code = 15  // 0xfff
	PingClockTime            Code = 21  // 0x15ff
	CreateGame               Code = 28  // 0x1cff
	ClientHostAndUsername    Code = 30  // 0x1eff
	JoinGame                 Code = 34  // 0x22ff
	ClientAuthentication     Code = 41  // 0x29ff
	CreateNewAccount         Code = 42  // 0x2aff
	UpdateCharacterInventory Code = 44  // 0x2cff
	GetCharacters            Code = 60  // 0x3cff
	DeleteCharacter          Code = 61  // 0x3dff
	GetCharacterInventory    Code = 68  // 0x44ff
	SelectGame               Code = 69  // 0x45ff
	ShowRanking              Code = 70  // 0x46ff
	HostMigration            Code = 71  // 0x47ff
	GetCharacterSpells       Code = 72  // 0x48ff
	UpdateCharacterSpells    Code = 73  // 0x49ff
	SelectCharacter          Code = 76  // 0x4cff
	CreateCharacter          Code = 92  // 0x5cff
	UpdateCharacterStats     Code = 108 // 0x6cff
)
