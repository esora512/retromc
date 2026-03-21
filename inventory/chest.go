package inventory

import "fmt"

// Treat single and double chest the same
type Chest struct {
	Size  uint16
	Items []Item
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

var chestRegistry map[string]*Chest

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
	delete(chestRegistry, key)
}

func PlaceChest(x, y, z int32) {
	chest := NewChest(27)
	if chestRegistry == nil {
		chestRegistry = make(map[string]*Chest)
	}
	chestRegistry[chestKey(x, y, z)] = &chest
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
		fmt.Printf("  slot %2d | id %-5d | count %d\n", i, item.TypeId, item.Count)
	}
	fmt.Println("=================")
}

func (c *Chest) IsInChest (slot int16) bool {
	return slot >= 0 && slot < int16(c.Size)
}
