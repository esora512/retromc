package player

import "github.com/leNicDev/retromc/packet"

type Item struct {
	TypeId int16
	Count  byte
	Uses   int16
}

func (item *Item) Serialize() []byte {
	writer := packet.NewPacketWriter()

	writer.WriteShort(uint16(item.TypeId))
	// Per protocol: count and uses are only present when the item is not empty (-1)
	if item.TypeId != -1 {
		writer.WriteByte(item.Count)
		writer.WriteShort(uint16(item.Uses))
	}

	return writer.Bytes()
}

func NewItem(typeId int16, count byte) Item {
	return Item{
		TypeId: typeId,
		Count:  count,
		Uses:   int16(0),
	}
}

func (item *Item) Half() {
	if item.Count == 1 {
		item.Count = 0
		item.TypeId = -1
	} else {
		item.Count = (item.Count + 1) / 2
	}
}
