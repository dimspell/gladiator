package model

import "fmt"

const (
	MagicSpellFirebolt = iota
	MagicSpellMagicArrow
	MagicSpellHealing
	MagicSpellCure
	_
	MagicSpellHealOther
	MagicSpellLighting
	MagicSpellFireball
	MagicSpellChainLighting
	_
	MagicSpellInfraVision
	MagicSpellElectricField
	MagicSpellJudgment
	MagicSpellBless
	MagicSpellHolyGrab
	MagicSpellDivineArmour
	MagicSpellLightOrb
	MagicSpellWindyArmour
	MagicSpellDivineGrab
	MagicSpellTurnUndead
	MagicSpellCircleOrb
	MagicSpellContinousOrb
	_
	MagicSpellCurse
	_
	MagicSpellDarkOrb
	MagicSpellVenom
	_
	MagicSpellGhostRedeption
	_
	MagicSpellConcetration
	MagicSpellGreatDarkOrb
	_
	MagicSpellSalamandar
	MagicSpellEfreet
	MagicSpellMellow
	MagicSpellUndine
	_ // Driad?
	MagicSpellGnome
	MagicSpellSylph
	MagicSpellDijini
)

func printSpells(spells []byte) {
	for i := 0; i < len(spells); i++ {
		fmt.Print(spells[i], " ")
	}
	fmt.Println()
	fmt.Println(len(spells))
}
