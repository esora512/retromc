package level

import (
	"math/rand"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/player"
)

type Containers struct {
	Chests     map[BlockKey]*inventory.Chest
	Dispensers map[BlockKey]*inventory.Dispenser
	Furnaces   map[BlockKey]*inventory.Furnace
}

const CHEST_SIZE = 27
const DOUBLE_CHEST_SIZE = 54
const CHEST_SHIFT = 18        // Move from 27-18 = 9
const DOUBLE_CHEST_SHIFT = 45 // Move from 54-45 = 9
const FURNACE_SIZE = 3

func containerKey(x, y, z, dim int32) BlockKey {
	return BlockKey{X: x, Y: byte(y), Z: z, Dim: dim}
}

func (w *World) SetCloseContainer(f func(w *World, pl *player.Player)) {
	w.closeContainer = f
}

func (w *World) PlaceDispenser(x, y, z, dim int32) bool {

	key := containerKey(x, y, z, dim)

	if _, ok := w.Containers.Dispensers[key]; ok {
		return false
	}

	dispenser := inventory.NewDispenser()
	dispenser.SetPosition(x, y, z)
	w.Containers.Dispensers[key] = dispenser
	return true
}

func (w *World) GetDispenser(x, y, z, dim int32) *inventory.Dispenser {
	key := containerKey(x, y, z, dim)

	if dispenser, ok := w.Containers.Dispensers[key]; ok {
		return dispenser
	}

	return nil
}

func (w *World) RemoveDispenser(x, y, z, dim int32) {
	key := containerKey(x, y, z, dim)
	delete(w.Containers.Dispensers, key)
}

func neighbourKeys(x, y, z, dim int32) [4]BlockKey {
	return [4]BlockKey{
		containerKey(x+1, y, z, dim),
		containerKey(x-1, y, z, dim),
		containerKey(x, y, z+1, dim),
		containerKey(x, y, z-1, dim),
	}
}

func (w *World) isChestBlock(k BlockKey) bool {
	ch, ok := w.PeekChunk(WorldToChunkCoord(k.X), WorldToChunkCoord(k.Z), k.Dim)
	if !ok || len(ch.Data) < chunkBlocksAmount {
		return true
	}
	return ch.GetBlock(int(k.X&15), int(k.Y), int(k.Z&15)).TypeId == byte(constants.Chest.Value)
}

func (w *World) adjacentChests(x, y, z, dim int32, checkBlocks bool) []*inventory.Chest {
	var out []*inventory.Chest
	for _, n := range neighbourKeys(x, y, z, dim) {
		if c, ok := w.Containers.Chests[n]; ok && (!checkBlocks || w.isChestBlock(n)) {
			out = append(out, c)
		}
	}
	return out
}

func attachChestHalf(c *inventory.Chest, x, y, z int32, items []inventory.Item) {
	c.Size = DOUBLE_CHEST_SIZE
	c.Items = append(c.Items[:CHEST_SIZE:CHEST_SIZE], items...)
	c.SetSecondPosition(x, y, z)
}

func emptyChestItems() []inventory.Item {
	items := make([]inventory.Item, CHEST_SIZE)
	for i := range items {
		items[i] = inventory.NewItem(-1, 0, 0)
	}
	return items
}

func (w *World) GetChest(x, y, z, dim int32) *inventory.Chest {
	key := containerKey(x, y, z, dim)
	if chest, ok := w.Containers.Chests[key]; ok {
		return chest
	}
	return nil
}

// ChestHalfItems returns the 27 slots that belong to the chest block at x, y, z.
func ChestHalfItems(chest *inventory.Chest, x, y, z int32) []inventory.Item {
	if chest.Size != DOUBLE_CHEST_SIZE {
		return chest.Items
	}
	if x == chest.SecondPosition.X && y == chest.SecondPosition.Y && z == chest.SecondPosition.Z {
		return chest.Items[CHEST_SIZE:]
	}
	return chest.Items[:CHEST_SIZE]
}

// RemoveChest removes the chest block at x, y, z and returns the items stored in that half.
func (w *World) RemoveChest(x, y, z, dim int32) []inventory.Item {
	key := containerKey(x, y, z, dim)
	chest, ok := w.Containers.Chests[key]
	if !ok {
		return nil
	}
	delete(w.Containers.Chests, key)

	removed := append([]inventory.Item(nil), ChestHalfItems(chest, x, y, z)...)
	if chest.Size == DOUBLE_CHEST_SIZE {
		atFirst := x == chest.Position.X && y == chest.Position.Y && z == chest.Position.Z
		if atFirst {
			chest.Items = append([]inventory.Item(nil), chest.Items[CHEST_SIZE:]...)
			chest.Position = chest.SecondPosition
		} else {
			chest.Items = append([]inventory.Item(nil), chest.Items[:CHEST_SIZE]...)
		}
		chest.Size = CHEST_SIZE
		chest.SecondPosition = inventory.ContainerPosition{}
	}
	return removed
}

// PlaceChest follows vanilla's rule: a chest may join exactly one adjacent single chest,
// and may not touch a double chest or two chests at once.
func (w *World) PlaceChest(x, y, z, dim int32) bool {
	key := containerKey(x, y, z, dim)
	if _, ok := w.Containers.Chests[key]; ok {
		w.RemoveChest(x, y, z, dim)
	}

	adj := w.adjacentChests(x, y, z, dim, true)
	if len(adj) > 1 {
		return false
	}
	if len(adj) == 1 {
		if adj[0].Size == DOUBLE_CHEST_SIZE {
			return false
		}
		attachChestHalf(adj[0], x, y, z, emptyChestItems())
		w.Containers.Chests[key] = adj[0]
		return true
	}

	chest := inventory.NewChest(CHEST_SIZE)
	chest.SetPosition(x, y, z)
	w.Containers.Chests[key] = &chest
	return true
}

// loadChest registers a chest read from disk, joining it with an already loaded neighbour
// half. Chests still held in memory are newer than disk and are kept as is.
func (w *World) loadChest(x, y, z, dim int32, items []inventory.Item) {
	key := containerKey(x, y, z, dim)
	if _, ok := w.Containers.Chests[key]; ok {
		return
	}
	if adj := w.adjacentChests(x, y, z, dim, false); len(adj) == 1 && adj[0].Size == CHEST_SIZE {
		attachChestHalf(adj[0], x, y, z, items)
		w.Containers.Chests[key] = adj[0]
		return
	}
	chest := inventory.Chest{Size: CHEST_SIZE, Items: items}
	chest.SetPosition(x, y, z)
	w.Containers.Chests[key] = &chest
}

// BreakContainer removes the container behind a destroyed block, closes it for anyone
// viewing it and scatters its contents like vanilla.
func (w *World) BreakContainer(x, y, z int32, oldType byte, dim int32) {
	var items []inventory.Item

	switch oldType {
	case byte(constants.Chest.Value):
		chest := w.GetChest(x, y, z, dim)
		if chest == nil {
			return
		}
		w.closeViewers(func(pl *player.Player) bool {
			return pl.InventoryType == player.ChestInventory && w.GetChest(pl.Chest.X, pl.Chest.Y, pl.Chest.Z, pl.Chest.Dim) == chest
		})
		items = w.RemoveChest(x, y, z, dim)
	case byte(constants.Furnace.Value), byte(constants.FurnaceLit.Value):
		furnace := w.GetFurnace(x, y, z, dim)
		if furnace == nil {
			return
		}
		w.closeViewers(func(pl *player.Player) bool {
			return pl.InventoryType == player.FurnaceInventory && w.GetFurnace(pl.Furnace.X, pl.Furnace.Y, pl.Furnace.Z, pl.Furnace.Dim) == furnace
		})
		items = furnace.Items[:]
		w.RemoveFurnace(x, y, z, dim)
	case byte(constants.Dispenser.Value):
		dispenser := w.GetDispenser(x, y, z, dim)
		if dispenser == nil {
			return
		}
		w.closeViewers(func(pl *player.Player) bool {
			return pl.InventoryType == player.DispenserInventory && w.GetDispenser(pl.Dispenser.X, pl.Dispenser.Y, pl.Dispenser.Z, pl.Dispenser.Dim) == dispenser
		})
		items = dispenser.Items[:]
		w.RemoveDispenser(x, y, z, dim)
	default:
		return
	}

	w.ScatterItems(items, float64(x), float64(y), float64(z), dim, 10)
}

func (w *World) closeViewers(viewing func(pl *player.Player) bool) {
	if w.closeContainer == nil {
		return
	}
	for _, pl := range w.Players {
		if pl.LoggedIn && viewing(pl) {
			w.closeContainer(w, pl)
		}
	}
}

// ScatterItems drops stacks around a block corner the way vanilla empties containers,
// splitting each stack into random chunks of 10-30.
func (w *World) ScatterItems(items []inventory.Item, x, y, z float64, dim, pickupDelay int32) {
	for _, stack := range items {
		if stack.TypeId == -1 || stack.Count == 0 {
			continue
		}
		offsetX := rand.Float64()*0.8 + 0.1
		offsetY := rand.Float64()*0.8 + 0.1
		offsetZ := rand.Float64()*0.8 + 0.1

		remaining := int(stack.Count)
		for remaining > 0 {
			n := min(rand.Intn(21)+10, remaining)
			remaining -= n

			const velocity = 0.05
			velX := rand.NormFloat64() * velocity
			velY := rand.NormFloat64()*velocity + 0.2
			velZ := rand.NormFloat64() * velocity

			id := w.AddDroppedItem(x+offsetX, y+offsetY, z+offsetZ, int32(stack.TypeId), byte(n), stack.Metadata, pickupDelay, dim, velX, velY, velZ)
			if d, ok := w.Entities[id].(*entities.DroppedItem); ok {
				d.MovementState.VelocityChanged = true
			}
		}
	}
}

func (w *World) PlaceFurnace(x, y, z, dim int32) bool {
	key := containerKey(x, y, z, dim)
	furnace := inventory.NewFurnace()
	furnace.SetPosition(x, y, z)
	furnace.Dim = dim
	w.Containers.Furnaces[key] = furnace
	return true
}

func (w *World) RemoveFurnace(x, y, z, dim int32) {
	key := containerKey(x, y, z, dim)
	delete(w.Containers.Furnaces, key)
}

func (w *World) GetFurnace(x, y, z, dim int32) *inventory.Furnace {
	key := containerKey(x, y, z, dim)
	return w.Containers.Furnaces[key]
}

func (w *World) GetAllFurnaces() []*inventory.Furnace {
	furnaces := make([]*inventory.Furnace, 0, len(w.Containers.Furnaces))
	for _, furnace := range w.Containers.Furnaces {
		furnaces = append(furnaces, furnace)
	}
	return furnaces
}

func (w *World) GetAllChests() []*inventory.Chest {
	chests := make([]*inventory.Chest, 0, len(w.Containers.Chests))
	for _, chest := range w.Containers.Chests {
		chests = append(chests, chest)
	}
	return chests
}

func (w *World) GetAllDispensers() []*inventory.Dispenser {
	dispensers := make([]*inventory.Dispenser, 0, len(w.Containers.Dispensers))
	for _, dispenser := range w.Containers.Dispensers {
		dispensers = append(dispensers, dispenser)
	}
	return dispensers
}
