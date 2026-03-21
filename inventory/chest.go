package inventory

const CHEST_SIZE = 27
const DOUBLE_CHEST_SIZE = 54

// Treat single and double chest the same
type Chest struct {
	Size  uint16
	Items []Item
}

func NewChest(size uint16) Inventory {
	inv := Inventory{
		Size:  size,
		Items: make([]Item, size),
	}
	for i := range inv.Items {
		inv.Items[i] = NewItem(-1, 0, 0)
	}
	return inv
}
