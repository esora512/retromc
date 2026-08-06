package player

import (
	"log"
	"net"
	"sync/atomic"

	"fmt"
	"strings"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/crafting"
	"github.com/leNicDev/retromc/inventory"
)

const (
	PLAYER_INVENTORY_SIZE = 45
)

type InventoryType int

type PlayerChest struct {
	X, Y, Z int32
}

type PlayerDispenser struct {
	X, Y, Z int32
}

type PlayerFurnace struct {
	X, Y, Z int32
}

const (
	PlayerInventory    InventoryType = 0
	WorkbenchInventory InventoryType = 1
	ChestInventory     InventoryType = 2
	DispenserInventory InventoryType = 3
	FurnaceInventory   InventoryType = 4
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

type ChunkCoord struct {
	X, Z int32
}

type ChunkSet map[string]*ChunkCoord

func (s ChunkSet) Has(key string) bool {
	_, ok := s[key]
	return ok
}

func (s ChunkSet) Set(key string, x, z int32) {
	s[key] = &ChunkCoord{X: x, Z: z}
}

func (pl *Player) GetLoggedIn() bool {
	return pl.LoggedIn
}

func (pl *Player) GetDim() int32 {
	return pl.Dimension
}

type Player struct {
	FallDistance float64
	Username     string
	EntityId     int
	Connection   net.Conn
	Inventory    inventory.Inventory
	SelectedItem SelectedItem
	HotbarSlot   int16
	HotbarLocked atomic.Bool // locked while a BlockPlacement is being processed

	// Last valid position — used for boundary rubber-banding.
	X, Y, Z              float64
	Vx, Vy, Vz           float64
	Lx, Ly, Lz           float64
	Stance               float64
	Yaw, Pitch           float32
	LYaw, LPitch         float32
	OnGround             bool
	Workbench            inventory.Workbench
	InventoryType        InventoryType
	Chest                PlayerChest
	Dispenser            PlayerDispenser
	Furnace              PlayerFurnace
	TimeStarted          bool
	LoggedIn             bool
	IsRiding             int32
	BelowZeroHeightCount int
	IsCreative           bool
	DebugBlock           bool

	SentChunks           ChunkSet
	LastChunkX           int32
	LastChunkZ           int32
	HasInitializedChunks bool
	Dimension            int32

	HP        int16
	OnlineFor int64
}

func (pl *Player) SetHP(hp int16) {
	pl.HP = hp
}

func (pl *Player) GetHP() int16 {
	return pl.HP
}

const (
	SpawnX      = 0.0
	SpawnY      = 70.0
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
		X:                    SpawnX,
		Y:                    SpawnY,
		Z:                    SpawnZ,
		Vx:                   0,
		Vy:                   0,
		Vz:                   0,
		Stance:               SpawnStance,
		Workbench:            *inventory.NewWorkbench(),
		InventoryType:        PlayerInventory,
		Chest:                PlayerChest{X: 0, Y: 0, Z: 0},
		Dispenser:            PlayerDispenser{X: 0, Y: 0, Z: 0},
		Furnace:              PlayerFurnace{X: 0, Y: 0, Z: 0},
		HotbarSlot:           36,
		IsRiding:             -1,
		BelowZeroHeightCount: 0,
		LYaw:                 0,
		LPitch:               0,
		IsCreative:           false,
		SentChunks:           make(ChunkSet),
		DebugBlock:           false,
		HP:                   20,
		Dimension:            0,
		OnlineFor:            0,
	}
}

func (pl *Player) SetPosition(x, y, z float64) {
	pl.X, pl.Y, pl.Z = x, y, z
}

func (pl *Player) IsPlayer() bool {
	return true
}

func (pl *Player) GetEntityId() int32 {
	return int32(pl.EntityId)
}

func (pl *Player) IsRideable() bool {
	return false
}

func (pl *Player) GetName() string {
	return pl.Username
}

func (pl *Player) GetPosition() (float64, float64, float64) {
	return pl.X, pl.Y, pl.Z
}

func (pl *Player) GivePlayer(input string) {
	args := strings.Split(input, " ")
	var amountInt int
	log.Printf("GivePlayer: %+v", args)
	if len(args) < 1 {
		return
	}
	if len(args) < 2 {
		amountInt = 1
	} else {
		_, err := fmt.Sscanf(args[1], "%d", &amountInt)
		if err != nil {
			return
		}
	}
	name := args[0]
	if amountInt <= 0 || amountInt > 64 {
		return
	}
	block := constants.GetBlockByName(name)
	if block.Value != -1 {
		pl.Inventory.AddItem(block.Value, block.Meta, byte(amountInt))
		return
	}
	item := constants.GetItemByName(name)
	if item.Value != -1 {
		if crafting.HasDurability(item.Value) {
			item.Meta = 0
		}
		pl.Inventory.AddItem(item.Value, item.Meta, byte(amountInt))
		return
	}
}
