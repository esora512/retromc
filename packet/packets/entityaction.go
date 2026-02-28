package packets

import (
	"github.com/leNicDev/retromc/packet"
)

type EntityActionInPacket struct {
	packet.Packet
	EntityId int
	ActionId byte
}

func ReadEntityActionInPacket(data *[]byte) EntityActionInPacket {
	reader := packet.PacketReader{
		Data: data,
	}

	packet := EntityActionInPacket{}
	packet.PacketId = reader.ReadPacketId()
	packet.EntityId = reader.ReadInt()
	packet.ActionId = reader.ReadByte()
	//log.Printf("Entity action: %+v", packet)

	return packet
}
