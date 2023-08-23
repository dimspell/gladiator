package model

import (
	"fmt"
)

type CharacterInventory struct {
	Backpack [63]InventoryItem // 7x9
	Belt     [6]InventoryItem  // 6
}

type InventoryItem struct {
	TypeId  byte
	ItemId  byte
	Unknown byte
}

func NewCharacterInventory(buf []byte) CharacterInventory {
	inv := CharacterInventory{}

	for i := 0; i < 63; i++ {
		slot := InventoryItem{
			TypeId:  buf[0+i*3],
			ItemId:  buf[1+i*3],
			Unknown: buf[2+i*3],
		}
		inv.Backpack[i] = slot
	}
	for i := 0; i < 6; i++ {
		slot := InventoryItem{
			TypeId:  buf[0+i*3],
			ItemId:  buf[1+i*3],
			Unknown: buf[2+i*3],
		}
		inv.Backpack[i] = slot
	}

	return inv
}

func (inv *CharacterInventory) Print() {
	i := 0
	for x := 0; x < 9; x++ {
		for y := 0; y < 7; y++ {
			fmt.Print(inv.Backpack[i], " \t")
			i++
		}
		fmt.Println()
	}

	for x := 0; x < 6; x++ {
		fmt.Print(inv.Belt[x], " \t")
	}
	fmt.Println()
}
