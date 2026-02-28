package packets

import (
	"log"

	"github.com/leNicDev/retromc/packet"
)

type PlayerLookInPacket struct {
	packet.Packet
	Yaw      float32
	Pitch    float32
	OnGround bool
}

func ReadPlayerLookInPacket(data *[]byte) PlayerLookInPacket {
	reader := packet.PacketReader{
		Data: data,
	}
	packet := PlayerLookInPacket{}
	packet.PacketId = reader.ReadPacketId()
	packet.Yaw = reader.ReadFloat32()
	packet.Pitch = reader.ReadFloat32()
	packet.OnGround = reader.ReadBool()
	log.Printf("Player look: %+v", packet)

	return packet
}
