package player

import (
	"log"

	"github.com/leNicDev/retromc/packet"

)

const (
	// Slots 0-8 are crafting output/grid and armour — not usable for item storage.
	// Slots 9-35 are the main inventory; 36-44 are the hotbar.
	StorageStart       = 9
	StorageEnd         = 44
	MaxStack           = 64
	HotbarStart        = 36
	HotbarEnd          = 44
	MainInventoryStart = 9
	MainInventoryEnd   = 35
)

type Inventory struct {
	Size  uint16
	Items []Item
}

func (inv *Inventory) GetCrafting2x2() [4]int16 {
	var arr [4]int16
	for i := range 4 {
		arr[i] = inv.PeekItem(int16(i + 1)).TypeId
	}
	return arr
}

// Serialize encodes all slots in window-items wire format.
// Each slot: itemId (short), and only if itemId != -1: count (byte) + uses (short)
func (inv *Inventory) Serialize() []byte {
	writer := packet.NewPacketWriter()
	for i := range inv.Items {
		writer.Write(inv.Items[i].Serialize())
	}
	return writer.Bytes()
}

// SetItem places an item directly into a slot, ignoring stack limits.
func (inv *Inventory) SetItem(slot int16, typeId int16, count byte) {
	if slot < 0 || int(slot) >= len(inv.Items) {
		log.Printf("SetItem: slot %d out of range", slot)
		return
	}
	inv.Items[slot] = NewItem(typeId, count)
}

// AddItem adds one of the given block type to the inventory (slots 9-44).
// It first tries to stack onto an existing partial stack of the same type,
// then falls back to the first empty slot.
// Returns the slot index that was updated, or -1 if the inventory is full.
func (inv *Inventory) AddItem(typeId int16) int16 {
	// Try to increment an existing partial stack.
	for i := HotbarStart; i <= HotbarEnd; i++ {
		item := &inv.Items[i]
		if item.TypeId == typeId && item.Count < MaxStack {
			item.Count++
			return int16(i)
		}
	}

	for i := MainInventoryStart; i <= MainInventoryEnd; i++ {
		item := &inv.Items[i]
		if item.TypeId == typeId && item.Count < MaxStack {
			item.Count++
			return int16(i)
		}
	}
	// No partial stack found — claim the first empty slot.
	for i := HotbarStart; i <= HotbarEnd; i++ {
		if inv.Items[i].TypeId == -1 {
			inv.Items[i] = NewItem(typeId, 1)
			return int16(i)
		}
	}

	for i := MainInventoryStart; i <= MainInventoryEnd; i++ {
		if inv.Items[i].TypeId == -1 {
			inv.Items[i] = NewItem(typeId, 1)
			return int16(i)
		}
	}
	return -1 // inventory full
}


// RemoveOneFromSlot decrements the count in a slot by one.
// When count reaches zero the slot is cleared to empty.
// Returns the slot index, or -1 if the slot was already empty.
func (inv *Inventory) RemoveOneFromSlot(slot int16) int16 {
	if slot < 0 || int(slot) >= len(inv.Items) {
		return -1
	}
	item := &inv.Items[slot]
	if item.TypeId == -1 {
		return -1
	}
	// Guard against underflow: Count is a byte, so decrementing 0 wraps to 255.
	if item.Count <= 1 {
		*item = NewItem(-1, 0)
	} else {
		item.Count--
	}
	return slot
}

func (inv *Inventory) PeekItem(slot int16) Item {
	return inv.Items[slot]
}

func (inv *Inventory) Move(sourceSlot, targetSlot int16) {
	inv.Items[targetSlot] = inv.Items[sourceSlot]
	inv.Items[sourceSlot] = NewItem(-1, 0)
}

func (inv *Inventory) Hold(slot int16) Item {
	item := inv.Items[slot]
	inv.Items[slot] = NewItem(-1, 0)
	return item
}

// TODO: Handle case where item count exceeds max stack size; leads to place & hold behaviour
func (inv *Inventory) Place(item Item, slot int16) {
	if inv.Items[slot].TypeId == item.TypeId {
		newCount := inv.Items[slot].Count + item.Count
		if newCount > MaxStack {
			item.Count = newCount - MaxStack
			inv.Items[slot].Count = MaxStack
		} else {
			inv.Items[slot].Count = newCount
		}
		return
	}
	inv.Items[slot] = item
}

func (inv *Inventory) PlaceOne(item *Item, slot int16) {
	if inv.Items[slot].TypeId == item.TypeId {
		inv.Items[slot].Count++
		item.Count--
	}
}

func (inv *Inventory) PlaceOneInEmpty(item *Item, slot int16) {
	if inv.Items[slot].TypeId == -1 {
		inv.Items[slot] = NewItem(item.TypeId, 0)
	}
	inv.PlaceOne(item, slot)
}

func (inv *Inventory) IsHotbarSlot(slot int16) bool {
	return slot >= HotbarStart && slot <= HotbarEnd
}

// Print logs every non-empty slot with its slot index, type ID, and count.
// Intended for debugging only.
func (inv *Inventory) Print() {
	log.Println("=== Inventory ===")
	for i, item := range inv.Items {
		if item.TypeId == -1 {
			continue
		}
		log.Printf("  slot %2d | id %-5d | count %d", i, item.TypeId, item.Count)
	}
	log.Println("=================")
}

func NewInventory(size uint16) Inventory {
	inv := Inventory{
		Size:  size,
		Items: make([]Item, size),
	}
	for i := range inv.Items {
		inv.Items[i] = NewItem(-1, 0)
	}
	return inv
}

func (inv *Inventory) RemoveCountFromInventory(slot int16, count byte) Item {
	if slot < 0 || int(slot) >= len(inv.Items) {
		return NewItem(-1, 0)
	}
	src := &inv.Items[slot]
	if src.TypeId == -1 {
		return NewItem(-1, 0)
	}
	if count >= src.Count {
		taken := *src
		*src = NewItem(-1, 0)
		return taken
	}
	taken := NewItem(src.TypeId, count)
	src.Count -= count
	return taken
}
