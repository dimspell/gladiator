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

// UDPCode identifies a UDP 6113 packet. Game packets carry type + session;
// raw packets carry type only.
type UDPCode byte

const (
	UDPRawStatusReply UDPCode = 14 // 14  raw 44 B
	UDPRawProbe       UDPCode = 27 // 27  raw 7 B answers 4 B ack

	UDPFullSnapshot   UDPCode = 1 // 28 B
	UDPSpawnRequest   UDPCode = 2 // var
	UDPSpawnAssign    UDPCode = 3 // 16 B
	UDPNameAnnounce   UDPCode = 4 // var
	UDPPeerAnnounce   UDPCode = 5 // 16 B
	UDPAppearance     UDPCode = 6 // 9 B
	UDPChat           UDPCode = 7 // var
	UDPDamage         UDPCode = 8 // 48 B
	UDPPeerListQuery  UDPCode = 11 // var
	UDPPeerList       UDPCode = 12 // 164 B
	UDPStatusRequest  UDPCode = 13 // var
	UDPDeathRespawn   UDPCode = 15 // 20 B
	UDPVisAck         UDPCode = 16 // 16 B
	UDPVisSync        UDPCode = 17 // 16 B
	UDPEntitySync     UDPCode = 18 // var
	UDPIPRelay        UDPCode = 19 // var, answers with TCP 20 (12 B)
	UDPMonsterDamage  UDPCode = 21 // 32 B
	UDPHitRoll        UDPCode = 22 // 16 B
	UDPStateOp        UDPCode = 23 // 24 B
	UDPCastStart      UDPCode = 24 // 24 B
	UDPCastEnd        UDPCode = 25 // 6 B
	UDPRelay1B        UDPCode = 26 // var
	UDPBulkSync       UDPCode = 29 // var 5+n*12
	UDPPlaceCreate    UDPCode = 30 // 24 B
	UDPPlaceRemove    UDPCode = 32 // 16 B
	UDPClassChange    UDPCode = 33 // var
	UDPMapSwitch      UDPCode = 34 // var
	UDPMonsterCompact UDPCode = 35 // 12 B
	UDPMonsterDamage2 UDPCode = 36 // 24 B
	UDPMonsterMove    UDPCode = 37 // 20 B
	UDPSpellCast      UDPCode = 38 // 12 B
	UDPMonsterHP      UDPCode = 39 // 12 B
	UDPMonsterAnim8   UDPCode = 40 // var
	UDPAttackState    UDPCode = 41 // var
	UDPDualTransfer   UDPCode = 48 // 24 B
	UDPAck32          UDPCode = 50 // var
	UDPSlot014        UDPCode = 51 // var
	UDPTargetedAction UDPCode = 52 // 36 B
	UDPFogQuery       UDPCode = 53 // 6 B
	UDPFogBitmap      UDPCode = 54 // 172 B
	UDPPerSlotInit    UDPCode = 55 // 16 B
	UDPSummonAtSelf   UDPCode = 56 // 16 B
	UDPSummonAtPeer   UDPCode = 57 // 16 B
	UDPPerUnitSync    UDPCode = 64 // 36 B
	UDPAvailQuery     UDPCode = 65 // 6 B
	UDPAvailBitmap    UDPCode = 66 // 38 B
	UDPBulkTrigger    UDPCode = 67 // 6 B
	UDPUnitKill       UDPCode = 80 // 8 B
	UDPReset8         UDPCode = 81 // 6 B
	UDPStageAdvance   UDPCode = 82 // 4 B
	UDPKick           UDPCode = 83 // 6 B
	UDPCancel         UDPCode = 84 // 6 B
)
