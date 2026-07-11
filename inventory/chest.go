package inventory

import (
	"fmt"
)

const CHEST_SIZE = 27
const DOUBLE_CHEST_SIZE = 54
const CHEST_SHIFT = 18        // Move from 27-18 = 9
const DOUBLE_CHEST_SHIFT = 45 // Move from 54-45 = 9

// Treat single and double chest the same
type Chest struct {
	Size           uint16
	Items          []Item
	Position       ContainerPosition
	SecondPosition ContainerPosition
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

func (c *Chest) SetItem(slot int16, typeId int16, count byte, metadata uint16) {
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

func (c *Chest) IsInChest(slot int16) bool {
	return slot >= 0 && slot < int16(c.Size)
}
