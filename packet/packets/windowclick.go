package packets

import (
	"log"

	"github.com/leNicDev/retromc/packet"
)

type WindowClickInPacket struct {
	WindowId     byte
	Slot         int16
	RightClick   byte
	ActionNumber int16
	Shift        bool
	ItemID       int16
	ItemCount    byte
	ItemUses     int16
}

func ReadWindowClickInPacket(data *[]byte) WindowClickInPacket {
	reader := packet.PacketReader{
		Data: data,
	}

	p := WindowClickInPacket{}
	_ = reader.ReadPacketId()
	p.WindowId = reader.ReadByte()
	p.Slot = int16(reader.ReadShort())
	p.RightClick = reader.ReadByte()
	p.ActionNumber = int16(reader.ReadShort())
	p.Shift = reader.ReadBool()
	p.ItemID = int16(reader.ReadShort())
	// count and uses are only present when the player is holding an item
	if p.ItemID != -1 {
		p.ItemCount = reader.ReadByte()
		p.ItemUses = int16(reader.ReadShort())
	}
	log.Printf("Slot: %d, ItemID: %d, ItemCount: %d, ItemUses: %d, Shift: %v", p.Slot, p.ItemID, p.ItemCount, p.ItemUses, p.Shift)
	return p
}
