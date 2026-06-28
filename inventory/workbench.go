package inventory

import "log"

type Workbench struct {
	Grid [9]Item
	Out  Item
}

func (wb *Workbench) IsCraftingSlot(slot int16) bool {
	return slot > 0 && slot <= 9
}

func (wb *Workbench) AddCount(slot int16, amount byte) {
	if slot < 1 || int(slot) > len(wb.Grid) {
		log.Printf("ChangeAmount: slot %d out of range", slot)
		return
	}
	wb.Grid[slot-1].Count += amount
}

func (wb *Workbench) SetEmpty(slot int16) {
	if slot < 1 || int(slot) > len(wb.Grid) {
		log.Printf("SetEmpty: slot %d out of range", slot)
		return
	}
	wb.Grid[slot-1] = EmptyItem()
}

func NewWorkbench() *Workbench {
	wb := Workbench{}
	for i := range wb.Grid {
		wb.Grid[i] = Item{-1, 0, 0}
	}
	return &wb
}

func (w *Workbench) ClearGrid() {
	for i := range w.Grid {
		w.Grid[i] = Item{-1, 0, 0}
	}
}

func EmptyItem() Item {
	return Item{-1, 0, 0}
}

func (w *Workbench) SetItem(slot int16, typeId int16, count byte, metadata uint16) {
	if slot < 1 || int(slot) > len(w.Grid) {
		log.Printf("SetItem: slot %d out of range", slot)
		return
	}
	item := NewItem(typeId, count, metadata)
	w.Grid[slot-1] = item
}

func (w *Workbench) PeekItem(slot int16) Item {
	loc := slot - 1
	if loc > 8 || loc < 0 {
		return EmptyItem()
	}
	return w.Grid[slot-1]
}


func (w *Workbench) GetGrid() [9]int16 {
	var arr [9]int16
	for i := range 9 {
		arr[i] = w.PeekItem(int16(i + 1)).TypeId
	}
	return arr
}

// RemoveOne decrements the count in a grid slot by one.
// When count reaches zero the slot is cleared to empty.
// Slot is 1-indexed (1-9).
func (w *Workbench) RemoveOne(slot int16) int16 {
	loc := slot - 1
	if loc < 0 || loc > 8 {
		return -1
	}
	item := &w.Grid[loc]
	if item.TypeId == -1 {
		return -1
	}
	if item.Count <= 1 {
		*item = EmptyItem()
	} else {
		item.Count--
	}
	return slot
}

func (w *Workbench) Print() {
	log.Println("=== Workbench ===")
	for i, item := range w.Grid {
		if item.TypeId == -1 {
			continue
		}
		log.Printf("  slot %2d | id %-5d | count %d | meta %d", i, item.TypeId, item.Count, item.Metadata)
	}
	log.Println("=================")
}
