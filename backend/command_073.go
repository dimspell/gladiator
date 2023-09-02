package backend

import "github.com/dispel-re/dispel-multi/model"

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

type UpdateCharacterSpellsRequest []byte
