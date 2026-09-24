package inventory

import (
	"log"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/packet"
)

const (
	// Slots 0-8 are crafting output/grid and armour — not usable for item storage.
	// Slots 9-35 are the main inventory; 36-44 are the hotbar.
	StorageStart       = 9
	StorageEnd         = 44
	HotbarStart        = 36
	HotbarEnd          = 44
	MainInventoryStart = 9
	MainInventoryEnd   = 35

	ChestStart = 0
	ChestEnd   = 26
)

func MaxStack(typeId int16) int {
	switch {
	case !IsStackable(typeId):
		return 1
	case typeId == constants.Snowball.Value, typeId == constants.Egg.Value:
		return 16
	case typeId == constants.Cookie.Value:
		return 8
	default:
		return 64
	}
}

type ContainerPosition struct {
	X, Y, Z int32
}

type ItemContainer interface {
	PeekItem(slot int16) Item
	SetItem(slot int16, typeId int16, count byte, metadata uint16)
	RemoveOne(slot int16) int16
	AddCount(slot int16, amount byte)
	SetEmpty(slot int16)
}

func MoveFromSourceToTargetContainer(sourceContainer, targetContainer ItemContainer, sourceSlot int16, regionStart, regionEnd int) bool {
	source := sourceContainer.PeekItem(sourceSlot)
	if source.TypeId == -1 {
		return false
	}

	step := 1
	if regionStart > regionEnd {
		step = -1
	}

	for i := regionStart; ; i += step {
		if targetContainer.PeekItem(int16(i)).TypeId == -1 {
			targetContainer.SetItem(int16(i), source.TypeId, source.Count, source.Metadata)
			sourceContainer.SetEmpty(sourceSlot)
			return true
		}
		if t := targetContainer.PeekItem(int16(i)); t.TypeId == source.TypeId && t.Metadata == source.Metadata &&
			int(t.Count)+int(source.Count) <= MaxStack(source.TypeId) {
			targetContainer.AddCount(int16(i), source.Count)
			sourceContainer.SetEmpty(sourceSlot)
			return true

		}
		if i == regionEnd {
			break
		}
	}

	return false
}
func (inv *Inventory) AddCount(slot int16, amount byte) {
	if slot < 0 || int(slot) >= len(inv.Items) {
		//log.Printf("ChangeAmount: slot %d out of range", slot)
		return
	}
	inv.Items[slot].Count += amount
}

func (inv *Inventory) SetEmpty(slot int16) {
	if slot < 0 || int(slot) >= len(inv.Items) {
		//log.Printf("SetEmpty: slot %d out of range", slot)
		return
	}
	inv.Items[slot] = NewItem(-1, 0, 0)
}

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
func (inv *Inventory) SetItem(slot int16, typeId int16, count byte, metadata uint16) {
	if slot < 0 || int(slot) >= len(inv.Items) {
		//log.Printf("SetItem: slot %d out of range", slot)
		return
	}
	inv.Items[slot] = NewItem(typeId, count, metadata)
}

func (inv *Inventory) AddItemHotbarFromRightToLeft(typeId int16, metadata uint16, count byte) bool {
	for i := HotbarEnd; i >= HotbarStart; i-- {
		item := &inv.Items[i]
		maxStack := MaxStack(item.TypeId)
		if item.TypeId == typeId && item.Metadata == metadata && int(item.Count)+int(count) <= maxStack {
			item.Count += count
			return true
		}
	}
	for i := HotbarEnd; i >= HotbarStart; i-- {
		if inv.Items[i].TypeId == -1 {
			inv.Items[i] = NewItem(typeId, count, metadata)
			return true
		}
	}
	return false
}


func (inv *Inventory) MergeItem(it *Item, reverse bool, start, end int) (touched []int16) {
	next := func(i int) int {
		if reverse {
			return i - 1
		}
		return i + 1
	}
	first := start
	if reverse {
		first = end
	}
	inRange := func(i int) bool { return i >= start && i <= end }

	if IsStackable(it.TypeId) {
		for i := first; inRange(i) && it.Count > 0; i = next(i) {
			slot := &inv.Items[i]
			if slot.TypeId != it.TypeId || slot.Metadata != it.Metadata {
				continue
			}
			space := MaxStack(slot.TypeId) - int(slot.Count)
			if space <= 0 {
				continue
			}
			move := min(space, int(it.Count))
			slot.Count += byte(move)
			it.Count -= byte(move)
			touched = append(touched, int16(i))
		}
	}
	if it.Count == 0 {
		return touched
	}
	for i := first; inRange(i); i = next(i) {
		if inv.Items[i].TypeId == -1 {
			inv.Items[i] = *it
			it.Count = 0
			return append(touched, int16(i))
		}
	}
	return touched
}

func (inv *Inventory) canTopUp(it Item, start, end int) bool {
	if !IsStackable(it.TypeId) {
		return false
	}
	for i := start; i <= end; i++ {
		slot := inv.Items[i]
		if slot.TypeId == it.TypeId && slot.Metadata == it.Metadata && int(slot.Count) < MaxStack(slot.TypeId) {
			return true
		}
	}
	return false
}

func (inv *Inventory) PickupItem(it *Item) []int16 {
	var touched []int16
	if inv.canTopUp(*it, MainInventoryStart, MainInventoryEnd) {
		touched = inv.MergeItem(it, false, MainInventoryStart, MainInventoryEnd)
		if it.Count == 0 {
			return touched
		}
	}
	touched = append(touched, inv.MergeItem(it, false, HotbarStart, HotbarEnd)...)
	if it.Count > 0 {
		touched = append(touched, inv.MergeItem(it, false, MainInventoryStart, MainInventoryEnd)...)
	}
	return touched
}

// AddItem stores the whole stack or nothing, using the pickup order.
// Returns a slot that was updated, or -1 if it didn't fit.
func (inv *Inventory) AddItem(typeId int16, metadata uint16, count byte) int16 {
	return inv.addAllOrNothing(NewItem(typeId, count, metadata), func(it *Item) []int16 { return inv.PickupItem(it) })
}

// AddItemReverse stores the whole stack in [start,end] walking backwards (shift-click craft/chest order).
func (inv *Inventory) AddItemReverse(typeId int16, metadata uint16, count byte, start, end int) int16 {
	return inv.addAllOrNothing(NewItem(typeId, count, metadata), func(it *Item) []int16 { return inv.MergeItem(it, true, start, end) })
}

func (inv *Inventory) addAllOrNothing(it Item, store func(*Item) []int16) int16 {
	backup := append([]Item(nil), inv.Items...)
	touched := store(&it)
	if it.Count > 0 || len(touched) == 0 {
		copy(inv.Items, backup)
		return -1
	}
	return touched[len(touched)-1]
}

// IsArmorFor reports whether typeId can go in armor slot 5 (helmet) .. 8 (boots)
func IsArmorFor(typeId int16, slot int16) bool {
	if typeId < constants.LeatherCap.Value || typeId > constants.GoldBoots.Value {
		return false
	}
	return int16((typeId-constants.LeatherCap.Value)%4) == slot-5
}

// RemoveOne decrements the count in a slot by one.
// When count reaches zero the slot is cleared to empty.
// Returns the slot index, or -1 if the slot was already empty.
func (inv *Inventory) RemoveOne(slot int16) int16 {
	if slot < 0 || int(slot) >= len(inv.Items) {
		return -1
	}
	item := &inv.Items[slot]
	if item.TypeId == -1 {
		return -1
	}
	// Guard against underflow: Count is a byte, so decrementing 0 wraps to 255.
	if item.Count <= 1 {
		*item = NewItem(-1, 0, 0)
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
	inv.Items[sourceSlot] = NewItem(-1, 0, 0)
}

func MoveFromWorkbenchToInventory(wb *Workbench, inv *Inventory, sourceSlot, targetSlot int16) {
	if sourceSlot > 8 && sourceSlot < 1 {
		return
	}

	inv.Items[targetSlot] = wb.Grid[sourceSlot-1]
	wb.Grid[sourceSlot-1] = NewItem(-1, 0, 0)
}

func (inv *Inventory) Hold(slot int16) Item {
	item := inv.Items[slot]
	inv.Items[slot] = NewItem(-1, 0, 0)
	return item
}

// TODO: Handle case where item count exceeds max stack size; leads to place & hold behaviour
func (inv *Inventory) Place(item Item, slot int16) {
	if inv.Items[slot].TypeId == item.TypeId && inv.Items[slot].Metadata == item.Metadata {
		maxStack := byte(MaxStack(item.TypeId))
		newCount := inv.Items[slot].Count + item.Count
		if newCount > maxStack {
			item.Count = newCount - maxStack
			inv.Items[slot].Count = maxStack
		} else {
			inv.Items[slot].Count = newCount
		}
		return
	}
	inv.Items[slot] = item
}

func (inv *Inventory) PlaceOne(item *Item, slot int16) {
	if inv.Items[slot].TypeId == item.TypeId && inv.Items[slot].Metadata == item.Metadata {
		inv.Items[slot].Count++
		item.Count--
	}
}

func (inv *Inventory) PlaceOneInEmpty(item *Item, slot int16) {
	if inv.Items[slot].TypeId == -1 {
		inv.Items[slot] = NewItem(item.TypeId, 0, item.Metadata)
	}
	inv.PlaceOne(item, slot)
}

func (inv *Inventory) IsCraftingSlot(slot int16) bool {
	return slot >= 0 && slot <= 5
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
		log.Printf("  slot %2d | id %-5d | count %d | meta %d", i, item.TypeId, item.Count, item.Metadata)
	}
	log.Println("=================")
}

func NewInventory(size uint16) Inventory {
	inv := Inventory{
		Size:  size,
		Items: make([]Item, size),
	}
	for i := range inv.Items {
		inv.Items[i] = NewItem(-1, 0, 0)
	}
	return inv
}

func (inv *Inventory) RemoveCountFromInventory(slot int16, count byte) Item {
	if slot < 0 || int(slot) >= len(inv.Items) {
		return NewItem(-1, 0, 0)
	}
	src := &inv.Items[slot]
	if src.TypeId == -1 {
		return NewItem(-1, 0, 0)
	}
	if count >= src.Count {
		taken := *src
		*src = NewItem(-1, 0, 0)
		return taken
	}
	taken := NewItem(src.TypeId, count, src.Metadata)
	src.Count -= count
	return taken
}
