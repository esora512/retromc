package packets

import (
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/inventory"

)

type SetSlotOutPacket struct {
	WindowId byte
	Slot     int16
	Item     inventory.Item
}

func (p *SetSlotOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()

	writer.WriteByte(packet.SetSlot)  // write packet id
	writer.WriteByte(p.WindowId)      // write window id
	writer.WriteShort(uint16(p.Slot)) // write slot
	writer.Write(p.Item.Serialize())  // write item

	return writer.Bytes()
}
