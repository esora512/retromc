package packets

import (
	"log"

	"github.com/leNicDev/retromc/packet"
)

type MineInPacket struct {
	packet.Packet
	Status byte
	X      int32
	Y      byte
	Z      int32
	Face   byte
}

func ReadMineInPacket(data *[]byte) MineInPacket {
	reader := packet.PacketReader{
		Data: data,
	}

	packet := MineInPacket{}
	packet.PacketId = reader.ReadPacketId()
	packet.Status = reader.ReadByte()
	packet.X = reader.ReadInt32()
	packet.Y = reader.ReadByte()
	packet.Z = reader.ReadInt32()
	packet.Face = reader.ReadByte()
	log.Printf("Mine: %+v", packet)

	return packet
}
