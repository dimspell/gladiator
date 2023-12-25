package model

import (
	"reflect"
	"testing"
)

func TestCharacterInventory_ToBytes(t *testing.T) {
	tests := []struct {
		name string
		inv  CharacterInventory
		want []byte
	}{
		{
			name: "Test 1",
			inv: CharacterInventory{
				Backpack: [63]InventoryItem{
					{
						TypeId:  1,
						ItemId:  2,
						Unknown: 3,
					},
					{
						TypeId:  4,
						ItemId:  5,
						Unknown: 6,
					},
				},
				Belt: [6]InventoryItem{
					{
						TypeId:  7,
						ItemId:  8,
						Unknown: 9,
					},
					{
						TypeId:  10,
						ItemId:  11,
						Unknown: 12,
					},
				},
			},
			want: []byte{
				1, 2, 3, 4, 5, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 7, 8, 9, 10, 11, 12, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.inv.ToBytes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CharacterInventory.ToBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}
