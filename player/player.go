package player

import (
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
