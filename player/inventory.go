package player

import "github.com/leNicDev/retromc/packet"

type Inventory struct {
	Size  uint16
	Items []Item
}

// serialize the inventory data
// see Window items (0x68): https://wiki.vg/index.php?title=Protocol&oldid=483
// Each slot: itemId (short), and only if itemId != -1: count (byte) + uses (short)
func (inv *Inventory) Serialize() []byte {
	writer := packet.NewPacketWriter()

	for i := range inv.Items {
		writer.Write(inv.Items[i].Serialize())
	}

	return writer.Bytes()
}

func NewInventory(size uint16) Inventory {
	inv := Inventory{
		Size:  size,
		Items: make([]Item, size),
	}

	// fill inventory with empty slots
	for i := range inv.Items {
		inv.Items[i] = NewItem(-1, 1)
	}

	return inv
}
