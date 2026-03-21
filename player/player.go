package player

import (
	"log"
	"net"
	"sync/atomic"

	"github.com/leNicDev/retromc/inventory"
)

const (
	PLAYER_INVENTORY_SIZE = 45
)

type InventoryType int

type PlayerChest struct {
	X, Y, Z int32
}

const (
	PlayerInventory    InventoryType = 0
	WorkbenchInventory InventoryType = 1
	ChestInventory     InventoryType = 2
)

type SelectedItem struct {
	Selected     bool
	Item         inventory.Item
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

func (si *SelectedItem) Clear() {
	si.Selected = false
	si.Item = inventory.Item{TypeId: -1, Count: 0, Metadata: 0}
	si.Slot = -1
	si.ActionNumber = -1
	si.RightClick = false
}

func (si *SelectedItem) SetItem(item inventory.Item, slot int16, actionNumber int16, rightClick bool) {
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
	Inventory    inventory.Inventory
	SelectedItem SelectedItem
	HotbarSlot   int16
	HotbarLocked atomic.Bool // locked while a BlockPlacement is being processed

	// Last valid position — used for boundary rubber-banding.
	X, Y, Z       float64
	Stance        float64
	Yaw, Pitch    float32
	OnGround      bool
	Workbench     inventory.Workbench
	InventoryType InventoryType
	Chest         PlayerChest
}

const (
	SpawnX      = 0.0
	SpawnY      = 64.0
	SpawnZ      = 0.0
	SpawnStance = SpawnY + 2
)

func NewPlayer(conn net.Conn) *Player {
	return &Player{
		Connection: conn,
		Inventory:  inventory.NewInventory(PLAYER_INVENTORY_SIZE),
		SelectedItem: SelectedItem{
			Selected: false,
			Item: inventory.Item{
				TypeId: -1,
				Count:  0,
			},
			Slot:         -1,
			ActionNumber: -1,
		},
		X:             SpawnX,
		Y:             SpawnY,
		Z:             SpawnZ,
		Stance:        SpawnStance,
		Workbench:     *inventory.NewWorkbench(),
		InventoryType: PlayerInventory,
		Chest:         PlayerChest{X: 0, Y: 0, Z: 0},
	}
}
