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

	log.Printf("Adding item %d to main inventory", typeId)

	for i := StorageStart; i <= StorageEnd; i++ {
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

	for i := StorageStart; i <= StorageEnd; i++ {
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
		log.Printf("RemoveOneFromSlot: slot %d out of range", slot)
		return -1
	}
	item := &inv.Items[slot]
	if item.TypeId == -1 {
		log.Printf("RemoveOneFromSlot: slot %d is empty", slot)
		return -1
	}
	item.Count--
	if item.Count == 0 {
		*item = NewItem(-1, 1)
	}
	return slot
}

// FindFirstSlotWith returns the first storage slot (9-44) that holds at least
// one item of the given type. Returns -1 if none found.
func (inv *Inventory) FindFirstSlotWith(typeId int16) int16 {
	for i := StorageStart; i <= StorageEnd; i++ {
		if inv.Items[i].TypeId == typeId && inv.Items[i].Count > 0 {
			return int16(i)
		}
	}
	return -1
}

func (inv *Inventory) Swap(slot1, slot2 int16) {
	inv.Items[slot1], inv.Items[slot2] = inv.Items[slot2], inv.Items[slot1]
}

func NewInventory(size uint16) Inventory {
	inv := Inventory{
		Size:  size,
		Items: make([]Item, size),
	}
	for i := range inv.Items {
		inv.Items[i] = NewItem(-1, 1)
	}
	return inv
}
