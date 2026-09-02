package packet

// UDP 6113 packets — all types known from DispelMulti_debug.c
// FUN_004040FB :3680 (game, session at 1-2) and FUN_00405D8B :4895 (raw, no session).

// Raw (no session, FUN_00405D8B :4895)
type RawStatusReply struct { // 0x0E 44 B
	Data [44]byte
}
type RawPeerProbe struct { // 0x1B 7 B probe, answered with a 4 B ack
	IP   [4]byte
	Port uint16
}

// IsRaw reports if a 6113 datagram is raw (no session) vs game.
func IsRaw(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	return b[0] == 0x0E || b[0] == 0x1B
}

// ValidateGame checks the session of game packets (type + reserved byte +
// session). Raw packets are always valid (no session check).
func ValidateGame(b []byte, session uint16) bool {
	if len(b) < 4 {
		return false
	}
	if IsRaw(b) {
		return true
	}
	got := uint16(b[2]) | uint16(b[3])<<8
	return got == session
}

// GameStartTracker counts 0x0E arrivals for the host.
type GameStartTracker struct {
	Expected int
	Arrived  int
}

func (t *GameStartTracker) Add(b []byte) bool {
	if len(b) > 0 && b[0] == 0x0E {
		t.Arrived++
	}
	return t.Expected > 0 && t.Arrived >= t.Expected
}

// Game (type 1 B + session 2 B + payload)
type Snapshot struct { // 0x01 28 B :3738
	Class uint8  // pkt[0x14]
	Tile  uint32 // pkt[8]
	SubTile uint32 // pkt[0xC]
	Flags uint32 // pkt[0x10]
	Stats [6]uint16
}

type SpawnRequest struct { // 0x02 var
	ClassCfg uint8
}
type SpawnAssign struct { // 0x03 16 B :3769
	Tile uint32 // DAT_0047F988
}
type NameAnnounce struct { // 0x04 var
	Name [15]byte // NUL+pad to 15 -> slot+0xE48
}
type PeerAnnounce struct { // 0x05 16 B :3794
	Tile uint32
	IP   [4]byte
}
type Appearance struct { // 0x06 9 B :3900
	Data [6]byte
}
type Chat struct { // 0x07 var 9+text
	Text string // NUL MBCS
}
type Damage struct { // 0x08 48 B :3909
	TargetTile uint32
	Value      uint32
	AttackerIP [4]byte
}
type PeerListQuery struct { // 0x0B var :3941
}
type PeerList struct { // 0x0C 164 B :3944
	OwnIP   [4]byte
	PeerIPs [7][4]byte
	Flags   [7]byte // 2 = connected
	OwnName [15]byte
	PeerNames [7][15]byte
}
type StatusRequest struct { // 0x0D var :3947 -> 0x0E
}
type DeathRespawn struct { // 0x0F 20 B :3876
	RespawnTile uint32
	Class       byte
}
type VisAck struct { // 0x10 16 B :3969 -> 0x11
	Tile uint32
	IP   [4]byte
}
type VisSync struct { // 0x11 16 B :3994
	Class byte // pkt[3]
	Tile  uint32 // pkt[4]
}
type EntitySync struct { // 0x12 var :4015
	Class byte
	Tile  uint32
	Value uint16 // v for XOR 0x3FF
}
type IPRelay struct { // 0x13 var :4034 -> TCP 0x14 12 B
	Index byte // pkt[3] -> DAT_00466F48
}
type MonsterDamage struct { // 0x15 32 B :4046
	StorageIdx uint32 // pkt[1]-10
	Tile       uint32 // pkt[3]
	Class      byte   // pkt[4]
	AttackerIP [4]byte // pkt[5]
}
type HitRoll struct { // 0x16 16 B :4074
	DstTile   uint32 // pkt[1]
	SrcTile   uint32 // pkt[2]
	HitResult byte   // pkt[0x12]
}
type StateOp struct { // 0x17 24 B :4099
	TargetIdx byte // pkt[0xC]
	Action    int16 // pkt[0x10] 0=idle 8, -1=miss
}
type CastStart struct { // 0x18 24 B :4137
	SpellID byte // pkt[0xD]
	TargetTile uint32
}
type CastEnd struct { // 0x19 6 B :4157
	SubAction byte // pkt[3]
}
type Relay1B struct { // 0x1A var :4174 -> 0x1B
}
type BulkSync struct { // 0x1D var 5+n*12 :4195
	Count byte // pkt[1]
	Entries []BulkEntry // 12 B each {key u32, id u16, kind u8}
}
type BulkEntry struct {
	Key  uint32
	Id   uint16
	Kind byte // 1/2/4
}
type PlaceCreate struct { // 0x1E 24 B :4218
	Tile  uint32 // pkt[0xC]
	Class byte   // pkt[4]
	Key   uint32
	Id    uint16
	Kind  byte
}
type PlaceRemove struct { // 0x20 16 B :4230
	Key  uint32
	Kind byte
}
type ClassChange struct { // 0x21 var :4233
	Class byte // pkt[1]
	Tile  uint32 // pkt[8]
}
type MapSwitch struct { // 0x22 var :4250
	Tile uint32 // pkt[4]
}
type MonsterCompact struct { // 0x23 12 B :4261
	Idx byte // pkt[1]
}
type MonsterDamage2 struct { // 0x24 24 B :4275
	StorageIdx uint32
	Tile       uint32
	Class      byte
	AttackerIP [4]byte
}
type MonsterMove struct { // 0x25 20 B :4303
	StorageIdx uint32
	Tile       uint32
	Class      byte
}
type SpellCast struct { // 0x26 12 B :4316
	SpellID byte // pkt[4]
	DstTile uint32 // pkt[8]
}
type MonsterHP struct { // 0x27 12 B :4341
	Class byte
	HP    uint16 // pkt[2]
	Tile  uint32 // pkt[3]
}
type MonsterAnim8 struct { // 0x28 var :4373
	Idx byte // pkt[5]-10
}
type AttackState struct { // 0x29 var :4377
	Action byte // pkt[5]
	Sub    byte // pkt[6]
}
type DualTransfer struct { // 0x30 24 B :4390
	SrcIdx byte // pkt[0xC]
	DstIdx byte // pkt[0x10]
	Action int16 // pkt[0x14]
}
type Ack32 struct { // 0x32 var :4459
}
type Slot014 struct { // 0x33 var :4471
	Value uint16 // pkt[4]
}
type TargetedAction struct { // 0x34 36 B :4483
	Action   uint32 // pkt[8]
	Sub      uint32 // pkt[0xC]
	TargetName string // pkt[0x14] NUL
}
type FogQuery struct { // 0x35 6 B :4514 -> 0x36
	Class byte // pkt[1]
}
type FogBitmap struct { // 0x36 172 B :4526
	Masks [5][32]byte // 256-bit per player
}
type PerSlotInit struct { // 0x37 16 B :4566
	Value uint32 // pkt[3]
	Param1 byte // pkt[1]
	Param2 byte // pkt[2]
}
type SummonAtSelf struct { // 0x38 16 B :4580
	Tile     uint32 // pkt[1]
	TargetIP [4]byte // pkt[3]
}
type SummonAtPeer struct { // 0x39 16 B :4595
	Tile uint32
}
type PerUnitSync struct { // 0x40 36 B :4607
	Value int16 // pkt[8] at -0x8958
}
type AvailQuery struct { // 0x41 6 B :4617 -> 0x42
	TargetIdx byte
}
type AvailBitmap struct { // 0x42 38 B :4629
	PopCount byte // +4
	Bits     [34]byte
}
type BulkTrigger struct { // 0x43 6 B :4662 -> 0x1D
	Kind byte // pkt[1]
}
type UnitKill struct { // 0x50 8 B :4676
	Idx byte // pkt[1]
}
type Reset8 struct { // 0x51 6 B :4703
	SubAction byte // pkt[5]
}
type StageAdvance struct { // 0x52 4 B :4719
}
type Kick struct { // 0x53 6 B :4733
	IP [4]byte // pkt[1]
}
type Cancel struct { // 0x54 6 B :4662
	SubAction byte // pkt[5]
}
