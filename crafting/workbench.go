package crafting

import "log"

type Item struct {
	TypeId      int16
	Count       byte
	SourceSlot  int16
	SourceCount *byte
}

type Workbench struct {
	Grid [9]Item
	Out  Item
}

func NewWorkbench() *Workbench {
	wb := Workbench{}
	for i := range wb.Grid {
		wb.Grid[i] = Item{-1, 0, -1, nil}
	}
	return &wb
}

func (w *Workbench) ClearGrid() {
	for i := range w.Grid {
		w.Grid[i] = Item{-1, 0, -1, nil}
	}
}

func NewItem(typeId int16, count byte, sourceSlot int16, sourceCount *byte) Item {
	return Item{
		TypeId:      typeId,
		Count:       count,
		SourceSlot:  sourceSlot,
		SourceCount: sourceCount,
	}
}

func (w *Workbench) SetGridItem(slot int16, item Item) {
	w.Grid[slot-1] = item
}

func (w *Workbench) GetGridItem(slot int16) Item {
	loc := slot - 1
	if loc > 8 || loc < 0 {
		return NewItem(-1, 0, -1, nil)
	}
	return w.Grid[slot-1]
}

func (w *Workbench) GetGrid() [9]int16 {
	var arr [9]int16
	for i := range 9 {
		arr[i] = w.GetGridItem(int16(i + 1)).TypeId
	}
	return arr
}

func (w *Workbench) Print() {
	log.Println("=== Workbench ===")
	for i, item := range w.Grid {
		if item.TypeId == -1 {
			continue
		}
		log.Printf("  slot %2d | id %-5d | count %d | source slot %d", i, item.TypeId, item.Count, item.SourceSlot)
	}
	log.Println("=================")
}
