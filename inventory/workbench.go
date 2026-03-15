package inventory

import "log"

type Workbench struct {
	Grid [9]Item
	Out  Item
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

func (w *Workbench) SetGridItem(slot int16, item Item) {
	w.Grid[slot-1] = item
}

func (w *Workbench) GetGridItem(slot int16) Item {
	loc := slot - 1
	if loc > 8 || loc < 0 {
		return EmptyItem()
	}
	return w.Grid[slot-1]
}

// GetGridItemPtr returns a pointer to a grid item for in-place modification.
// Slot is 1-indexed (1-9).
func (w *Workbench) GetGridItemPtr(slot int16) *Item {
	loc := slot - 1
	if loc > 8 || loc < 0 {
		return nil
	}
	return &w.Grid[loc]
}

func (w *Workbench) GetGrid() [9]int16 {
	var arr [9]int16
	for i := range 9 {
		arr[i] = w.GetGridItem(int16(i + 1)).TypeId
	}
	return arr
}

// RemoveOneFromGridSlot decrements the count in a grid slot by one.
// When count reaches zero the slot is cleared to empty.
// Slot is 1-indexed (1-9).
func (w *Workbench) RemoveOneFromGridSlot(slot int16) {
	loc := slot - 1
	if loc < 0 || loc > 8 {
		return
	}
	item := &w.Grid[loc]
	if item.TypeId == -1 {
		return
	}
	if item.Count <= 1 {
		*item = EmptyItem()
	} else {
		item.Count--
	}
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
