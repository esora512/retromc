package level

import (
	"github.com/leNicDev/retromc/inventory"
)

type Containers struct {
	Chests     map[BlockKey]*inventory.Chest
	Dispensers map[BlockKey]*inventory.Dispenser
	Furnaces   map[BlockKey]*inventory.Furnace
}

type ChestPlacement struct {
	AdjacentSlots  map[BlockKey]BlockKey
	ForbiddenSlots map[BlockKey]struct{}
}

const CHEST_SIZE = 27
const DOUBLE_CHEST_SIZE = 54
const CHEST_SHIFT = 18        // Move from 27-18 = 9
const DOUBLE_CHEST_SHIFT = 45 // Move from 54-45 = 9
const FURNACE_SIZE = 3

func containrKey(x, y, z int32) BlockKey {
	return BlockKey{X: x, Y: byte(y), Z: z}
}

func (w *World) PlaceDispenser(x, y, z int32) bool {

	key := containrKey(x, y, z)

	if _, ok := w.Containers.Dispensers[key]; ok {
		return false
	}

	dispenser := inventory.NewDispenser()
	dispenser.SetPosition(x, y, z)
	w.Containers.Dispensers[key] = dispenser
	return true
}

func (w *World) GetDispenser(x, y, z int32) *inventory.Dispenser {
	key := containrKey(x, y, z)

	if dispenser, ok := w.Containers.Dispensers[key]; ok {
		return dispenser
	}

	return nil
}

func (w *World) RemoveDispenser(x, y, z int32) {
	key := containrKey(x, y, z)
	delete(w.Containers.Dispensers, key)
}

func neighbourKeys(x, y, z int32) [4]BlockKey {
	return [4]BlockKey{
		containrKey(x+1, y, z),
		containrKey(x-1, y, z),
		containrKey(x, y, z+1),
		containrKey(x, y, z-1),
	}
}

func (w *World) registerDoubleAdjacentChest(x, y, z int32, excludePosition inventory.ContainerPosition) {
	for _, n := range neighbourKeys(x, y, z) {
		if n == containrKey(excludePosition.X, excludePosition.Y, excludePosition.Z) {
			continue
		}
		w.ChestPlacements.ForbiddenSlots[n] = struct{}{}
	}
}

func (w *World) unregisterDoubleAdjacentChest(x, y, z int32) {
	for _, n := range neighbourKeys(x, y, z) {
		delete(w.ChestPlacements.ForbiddenSlots, n)
	}
}

func (w *World) registerSingleAdjacentChest(x, y, z int32) {
	ownKey := containrKey(x, y, z)
	for _, n := range neighbourKeys(x, y, z) {
		w.ChestPlacements.AdjacentSlots[n] = ownKey
	}
}

func (w *World) unregisterSingleAdjacentChest(x, y, z int32) {
	for _, n := range neighbourKeys(x, y, z) {
		delete(w.ChestPlacements.AdjacentSlots, n)
	}
}

func (w *World) GetChest(x, y, z int32) *inventory.Chest {
	key := containrKey(x, y, z)
	if chest, ok := w.Containers.Chests[key]; ok {
		return chest
	}
	return nil
}

func (w *World) RemoveChest(x, y, z int32) {
	if w.Containers.Chests == nil {
		return
	}
	key := containrKey(x, y, z)
	chest := w.Containers.Chests[key]

	if chest.Size == DOUBLE_CHEST_SIZE {
		// Find surviving chest position
		var sPos inventory.ContainerPosition
		if x == chest.Position.X && y == chest.Position.Y && z == chest.Position.Z {
			sPos = chest.SecondPosition
		} else {
			sPos = chest.Position
		}

		chest.Size = CHEST_SIZE
		chest.Items = chest.Items[:CHEST_SIZE]

		delete(w.Containers.Chests, key)
		w.unregisterDoubleAdjacentChest(x, y, z)
		w.unregisterDoubleAdjacentChest(sPos.X, sPos.Y, sPos.Z)

		chest.Position = sPos
		chest.SecondPosition = inventory.ContainerPosition{}
		w.registerSingleAdjacentChest(sPos.X, sPos.Y, sPos.Z)
		return
	}

	delete(w.Containers.Chests, key)
	w.unregisterSingleAdjacentChest(x, y, z)
	for _, n := range neighbourKeys(x, y, z) {
		delete(w.ChestPlacements.ForbiddenSlots, n)
	}
	delete(w.ChestPlacements.AdjacentSlots, key)
	delete(w.ChestPlacements.ForbiddenSlots, key)
}

func (w *World) PlaceChest(x, y, z int32) bool {

	key := containrKey(x, y, z)
	if _, forbidden := w.ChestPlacements.ForbiddenSlots[key]; forbidden {
		return false
	}

	if neighbourKey, adjacent := w.ChestPlacements.AdjacentSlots[key]; adjacent {
		existingChest := w.Containers.Chests[neighbourKey]

		existingChest.Size = DOUBLE_CHEST_SIZE
		extra := make([]inventory.Item, CHEST_SIZE)
		for i := range extra {
			extra[i] = inventory.NewItem(-1, 0, 0)
		}
		existingChest.Items = append(existingChest.Items, extra...)
		// Point to same chest
		w.Containers.Chests[key] = existingChest

		// Single chest adjacency for ALLOWING placing new chests
		nx, ny, nz := existingChest.Position.X, existingChest.Position.Y, existingChest.Position.Z
		w.unregisterSingleAdjacentChest(nx, ny, nz)
		delete(w.ChestPlacements.AdjacentSlots, key)

		// Double chest adjaency for PREVENTING placing new chest
		w.registerDoubleAdjacentChest(nx, ny, nz, inventory.ContainerPosition{X: x, Y: y, Z: z})
		w.registerDoubleAdjacentChest(x, y, z, existingChest.Position)
		existingChest.SetSecondPosition(x, y, z)
		return true
	}

	chest := inventory.NewChest(CHEST_SIZE)
	chest.SetPosition(x, y, z)
	w.Containers.Chests[key] = &chest
	w.registerSingleAdjacentChest(x, y, z)
	return true
}

func (w *World) PlaceFurnace(x, y, z int32) bool {
	key := containrKey(x, y, z)
	furnace := inventory.NewFurnace()
	furnace.SetPosition(x, y, z)
	w.Containers.Furnaces[key] = furnace
	return true
}

func (w *World) RemoveFurnace(x, y, z int32) {
	key := containrKey(x, y, z)
	delete(w.Containers.Furnaces, key)
}

func (w *World) GetFurnace(x, y, z int32) *inventory.Furnace {
	key := containrKey(x, y, z)
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
