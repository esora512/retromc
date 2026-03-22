package inventory

import (
	"fmt"
	"log"
)

const CHEST_SIZE = 27
const DOUBLE_CHEST_SIZE = 54
const CHEST_SHIFT = 18        // Move from 27-18 = 9
const DOUBLE_CHEST_SHIFT = 45 // Move from 54-45 = 9

type ChestPosition struct {
	X int32
	Y int32
	Z int32
}

// Treat single and double chest the same
type Chest struct {
	Size           uint16
	Items          []Item
	Position       ChestPosition
	SecondPosition ChestPosition
}

func (c *Chest) SetPosition(x, y, z int32) {
	c.Position.X = x
	c.Position.Y = y
	c.Position.Z = z
}

func (c *Chest) SetSecondPosition(x, y, z int32) {
	c.SecondPosition.X = x
	c.SecondPosition.Y = y
	c.SecondPosition.Z = z
}

func (c *Chest) ShiftSlot(slot int16) int16 {
	if c.Size == CHEST_SIZE {
		return slot - CHEST_SHIFT
	}

	if c.Size == DOUBLE_CHEST_SIZE {
		return slot - DOUBLE_CHEST_SHIFT
	}
	return slot - CHEST_SHIFT
}

func NewChest(size uint16) Chest {
	inv := Chest{
		Size:  size,
		Items: make([]Item, size),
	}
	for i := range inv.Items {
		inv.Items[i] = NewItem(-1, 0, 0)
	}
	return inv
}

func (c *Chest) PeekItem(slot int16) Item {
	return c.Items[slot]
}

func (c *Chest) SetItem(slot int16, typeId int16, count byte, metadata byte) {
	c.Items[slot] = NewItem(typeId, count, metadata)
}

func (c *Chest) RemoveOne(slot int16) int16 {
	if c.Items[slot].TypeId == -1 {
		return -1
	}
	if c.Items[slot].Count <= 1 {
		c.Items[slot] = NewItem(-1, 0, 0)
	} else {
		c.Items[slot].Count--
	}
	return slot
}

func (c *Chest) AddCount(slot int16, amount byte) {
	c.Items[slot].Count += amount
}

func (c *Chest) SetEmpty(slot int16) {
	c.Items[slot] = NewItem(-1, 0, 0)
}

// Global chest management for all chests in the world
var chestRegistry map[string]*Chest
var adjacentSlots map[string]string
var forbiddenSlots map[string]struct{}

func initRegistries() {
	if chestRegistry == nil {
		chestRegistry = make(map[string]*Chest)
	}
	if adjacentSlots == nil {
		adjacentSlots = make(map[string]string)
	}
	if forbiddenSlots == nil {
		forbiddenSlots = make(map[string]struct{})
	}
}

func neighbourKeys(x, y, z int32) [4]string {
	return [4]string{
		chestKey(x+1, y, z),
		chestKey(x-1, y, z),
		chestKey(x, y, z+1),
		chestKey(x, y, z-1),
	}
}

func registerSingleAdjacentChest(x, y, z int32) {
	ownKey := chestKey(x, y, z)
	for _, n := range neighbourKeys(x, y, z) {
		adjacentSlots[n] = ownKey
	}
}

func unregisterSingleAdjacentChest(x, y, z int32) {
	for _, n := range neighbourKeys(x, y, z) {
		delete(adjacentSlots, n)
	}
}

func registerDoubleAdjacentChest(x, y, z int32, excludePosition ChestPosition) {
	for _, n := range neighbourKeys(x, y, z) {
		if n == chestKey(excludePosition.X, excludePosition.Y, excludePosition.Z) {
			continue
		}
		forbiddenSlots[n] = struct{}{}
	}
}

func unregisterDoubleAdjacentChest(x, y, z int32) {
	for _, n := range neighbourKeys(x, y, z) {
		delete(forbiddenSlots, n)
	}
}

func GetChest(x, y, z int32) *Chest {
	if chestRegistry == nil {
		chestRegistry = make(map[string]*Chest)
	}

	key := chestKey(x, y, z)

	if chest, ok := chestRegistry[key]; ok {
		return chest
	}
	return nil
}

func RemoveChest(x, y, z int32) {
	if chestRegistry == nil {
		return
	}
	key := chestKey(x, y, z)
	chest := chestRegistry[key]

	if chest.Size == DOUBLE_CHEST_SIZE {
		// Find surviving chest position
		var sPos ChestPosition
		if x == chest.Position.X && y == chest.Position.Y && z == chest.Position.Z {
			sPos = chest.SecondPosition
		} else {
			sPos = chest.Position
		}

		chest.Size = CHEST_SIZE
		chest.Items = chest.Items[:CHEST_SIZE]

		delete(chestRegistry, key)
		unregisterDoubleAdjacentChest(x, y, z)
		unregisterDoubleAdjacentChest(sPos.X, sPos.Y, sPos.Z)

		chest.Position = sPos
		chest.SecondPosition = ChestPosition{}
		registerSingleAdjacentChest(sPos.X, sPos.Y, sPos.Z)
		return
	}

	delete(chestRegistry, key)
	unregisterSingleAdjacentChest(x, y, z)
	for _, n := range neighbourKeys(x, y, z) {
		delete(forbiddenSlots, n)
	}
	delete(adjacentSlots, key)
	delete(forbiddenSlots, key)
}

func PlaceChest(x, y, z int32) bool {
	initRegistries()

	key := chestKey(x, y, z)
	if _, forbidden := forbiddenSlots[key]; forbidden {
		return false
	}

	if neighbourKey, adjacent := adjacentSlots[key]; adjacent {
		existingChest := chestRegistry[neighbourKey]

		existingChest.Size = DOUBLE_CHEST_SIZE
		extra := make([]Item, CHEST_SIZE)
		for i := range extra {
			extra[i] = NewItem(-1, 0, 0)
		}
		existingChest.Items = append(existingChest.Items, extra...)
		// Point to same chest
		chestRegistry[key] = existingChest

		// Single chest adjacency for ALLOWING placing new chests
		nx, ny, nz := existingChest.Position.X, existingChest.Position.Y, existingChest.Position.Z
		unregisterSingleAdjacentChest(nx, ny, nz)
		delete(adjacentSlots, key)

		// Double chest adjaency for PREVENTING placing new chest
		registerDoubleAdjacentChest(nx, ny, nz, ChestPosition{x, y, z})
		registerDoubleAdjacentChest(x, y, z, existingChest.Position)
		existingChest.SetSecondPosition(x, y, z)
		return true
	}

	chest := NewChest(CHEST_SIZE)
	chest.SetPosition(x, y, z)
	chestRegistry[key] = &chest
	registerSingleAdjacentChest(x, y, z)
	return true
}

func chestKey(x, y, z int32) string {
	return fmt.Sprintf("%d:%d:%d", x, y, z)
}

func (c *Chest) Print() {
	fmt.Println("=== Chest ===")
	for i, item := range c.Items {
		if item.TypeId == -1 {
			continue
		}
		fmt.Printf("  slot %2d | id %-5d | count %d | (%d %d %d)\n", i, item.TypeId, item.Count, c.Position.X, c.Position.Y, c.Position.Z)
	}
	fmt.Println("=================")
}

func PrintForbidden() {
	for key := range forbiddenSlots {
		log.Println("Forbidden", key)
	}
}

func (c *Chest) IsInChest(slot int16) bool {
	return slot >= 0 && slot < int16(c.Size)
}
