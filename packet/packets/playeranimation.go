package packets

import (
	"github.com/leNicDev/retromc/packet"
)

type PlayerAnimationInPacket struct {
	packet.Packet
	PlayerId int
	Animation byte
}

func ReadPlayerAnimationInPacket(data *[]byte) PlayerAnimationInPacket {
	reader := packet.PacketReader{
		Data: data,
	}

	packet := PlayerAnimationInPacket{}
	packet.PacketId = reader.ReadPacketId()
	packet.PlayerId = reader.ReadInt()
	packet.Animation = reader.ReadByte()
	//log.Printf("Player animation: %+v", packet)

	return packet
}
