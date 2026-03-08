package packets

import (
	"github.com/leNicDev/retromc/packet"
)

type EntityActionInPacket struct {
	packet.Packet
	EntityId int
	ActionId byte
}

func ReadEntityActionInPacket(reader packet.PacketReader) EntityActionInPacket {
	packet := EntityActionInPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.EntityId = reader.ReadInt()
	packet.ActionId = reader.ReadByte()
	//log.Printf("Entity action: %+v", packet)

	return packet
}
