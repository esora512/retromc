package inventory

import "github.com/leNicDev/retromc/packet"

type Item struct {
	TypeId   int16
	Count    byte
	Metadata byte
}

func (item *Item) Serialize() []byte {
	writer := packet.NewPacketWriter()

	writer.WriteShort(uint16(item.TypeId))
	// Per protocol: count and damage/metadata are only present when the item is not empty (-1)
	if item.TypeId != -1 {
		writer.WriteByte(item.Count)
		writer.WriteShort(uint16(item.Metadata))
	}

	return writer.Bytes()
}

func NewItem(typeId int16, count byte, metadata byte) Item {
	return Item{
		TypeId:   typeId,
		Count:    count,
		Metadata: metadata,
	}
}
