package crafting

type Item struct {
	TypeId int16
	Count  byte
	SourceSlot int16
}

type Worckbench struct {
	Grid [9]Item
	Out  Item
}

func NewWorckbench() *Worckbench {
	wb := Worckbench{}
	for i := range wb.Grid {
		wb.Grid[i] = Item{-1, 0, -1}
	}
	return &wb
}

func (w *Worckbench) SetGridItem(slot int16, item Item) {
	w.Grid[slot] = item
}

func (w *Worckbench) GetGridItem(slot int16) Item {
	return w.Grid[slot]
}

