package player

import (
	"log"
	"net"
	"sync/atomic"
)

const (
	PLAYER_INVENTORY_SIZE = 45
)

type SelectedItem struct {
	Selected     bool
	Item         Item
	Slot         int16
	ActionNumber int16
	RightClick   bool
}

func (si *SelectedItem) Print() {
	if !si.Selected {
		log.Println("No item selected")
		return
	}
	log.Printf("Item selected: %+v", si.Item)
}

func (si *SelectedItem) SetItem(item Item, slot int16, actionNumber int16, rightClick bool) {
	si.Item = item
	si.Slot = slot
	si.ActionNumber = actionNumber
	si.RightClick = rightClick
	si.Selected = true
}

type Player struct {
	Username     string
	EntityId     int
	Connection   net.Conn
	Inventory    Inventory
	SelectedItem SelectedItem
	HotbarSlot   int16
	HotbarLocked atomic.Bool // locked while a BlockPlacement is being processed
}

func NewPlayer(conn net.Conn) *Player {
	return &Player{
		Connection: conn,
		Inventory:  NewInventory(PLAYER_INVENTORY_SIZE),
		SelectedItem: SelectedItem{
			Selected: false,
			Item: Item{
				TypeId: -1,
				Count:  0,
				Uses:   0,
			},
			Slot:         -1,
			ActionNumber: -1,
		},
	}
}
