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

func (inv *Inventory) RemoveAllFromSlot(slot int16) int16 {
	if slot < 0 || int(slot) >= len(inv.Items) {
		return -1
	}
	inv.Items[slot] = NewItem(-1, 0)
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

func (inv *Inventory) FindFirstEmptySlotinHotbar() int16 {
	for i := HotbarStart; i <= HotbarEnd; i++ {
		if inv.Items[i].TypeId == -1 {
			return int16(i)
		}
	}
	return -1
}

func (inv *Inventory) FindFirstEmptySlotinMainInventory() int16 {
	for i := MainInventoryStart; i <= MainInventoryEnd; i++ {
		if inv.Items[i].TypeId == -1 {
			return int16(i)
		}
	}
	return -1
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

func (inv *Inventory) HoldHalf(slot int16) Item {
	item := inv.Items[slot]
	item.Count = (item.Count + 1) / 2
	inv.Items[slot].Count = (inv.Items[slot].Count + 1) / 2
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

func (inv *Inventory) SwapCounts(item Item, slot int16) {
	inv.Items[slot].Count, item.Count = item.Count, inv.Items[slot].Count
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

func (inv *Inventory) PlaceRest(item Item, slot int16) {
	if inv.Items[slot].Count+item.Count <= MaxStack {
		inv.Items[slot].Count += item.Count
		item.Count = 0
	} else {
		item.Count -= (MaxStack - inv.Items[slot].Count)
		inv.Items[slot].Count = MaxStack
	}
}

func (inv *Inventory) Drop(slot int16) {
	inv.Items[slot] = NewItem(-1, 0)
}

func (inv *Inventory) Swap(slot1, slot2 int16) {
	n := int16(len(inv.Items))
	if slot1 < 0 || slot1 >= n || slot2 < 0 || slot2 >= n {
		log.Printf("Swap: slot out of range (slot1=%d, slot2=%d, size=%d)", slot1, slot2, n)
		return
	}
	inv.Items[slot1], inv.Items[slot2] = inv.Items[slot2], inv.Items[slot1]
}

func (inv *Inventory) IsHotbarSlot(slot int16) bool {
	return slot >= HotbarStart && slot <= HotbarEnd
}

func (inv *Inventory) FindFirstNonStackSlotOfItemInHotbar(typeID int16) int16 {
	for i := HotbarStart; i <= HotbarEnd; i++ {
		if inv.Items[i].TypeId == typeID {
			return int16(i)
		}
	}
	return -1
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
