package packets

import (
	"github.com/leNicDev/retromc/packet"
)

type PlayerOnGroundInPacket struct {
	packet.Packet
	OnGround bool
}

func ReadPlayerOnGroundInPacket(data *[]byte) PlayerOnGroundInPacket {
	reader := packet.PacketReader{
		Data: data,
	}

	packet := PlayerOnGroundInPacket{}
	packet.PacketId = reader.ReadPacketId()
	packet.OnGround = reader.ReadBool()
	return packet
}
