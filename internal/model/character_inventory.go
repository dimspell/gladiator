package model

import (
	"fmt"
)

type CharacterInventory struct {
	Backpack [63]InventoryItem // 7x9
	Belt     [6]InventoryItem  // 6
}

type InventoryItem struct {
	TypeId       byte // wire id+1 (0 = empty slot)
	ItemId       byte // wire count+1 (0 = empty, wire 1 = count 1)
	PackedPos    byte // packedXY: (x+1)<<4 | (y+1), 0 = empty
}

// Decoded helpers — X 0-8, Y 0-6, Id = TypeId-1, Count = ItemId-1.
func (it InventoryItem) Id() byte    { return it.TypeId - 1 }
func (it InventoryItem) Count() byte { return it.ItemId - 1 }
func (it InventoryItem) X() byte     { return (it.PackedPos >> 4) - 1 }
func (it InventoryItem) Y() byte     { return (it.PackedPos & 0x0F) - 1 }
func (it InventoryItem) IsEmpty() bool {
	return it.TypeId == 0 && it.ItemId == 0 && it.PackedPos == 0
}

func NewCharacterInventory(buf []byte) CharacterInventory {
	inv := CharacterInventory{}
	if len(buf) < 207 {
		return inv
	}
	for i := 0; i < 63; i++ {
		off := i * 3
		it := InventoryItem{
			TypeId:    buf[off],
			ItemId:    buf[off+1],
			PackedPos: buf[off+2],
		}
		if it.IsEmpty() {
			continue
		}
		inv.Backpack[i] = it
	}
	for i := 0; i < 6; i++ {
		off := 189 + i*3
		it := InventoryItem{
			TypeId:    buf[off],
			ItemId:    buf[off+1],
			PackedPos: buf[off+2],
		}
		if it.IsEmpty() {
			continue
		}
		inv.Belt[i] = it
	}
	return inv
}

func (inv *CharacterInventory) ToBytes() []byte {
	out := make([]byte, 207)
	for i, item := range inv.Backpack {
		if item.IsEmpty() {
			continue
		}
		out[i*3] = item.TypeId
		out[i*3+1] = item.ItemId
		out[i*3+2] = item.PackedPos
	}
	for i, item := range inv.Belt {
		if item.IsEmpty() {
			continue
		}
		off := 189 + i*3
		out[off] = item.TypeId
		out[off+1] = item.ItemId
		out[off+2] = item.PackedPos
	}
	return out
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
