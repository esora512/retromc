package packets

import (
	"github.com/leNicDev/retromc/packet"
)

type PlaceInPacket struct {
	packet.Packet
	X      int32
	Y      byte
	Z      int32
	Face   byte
	ItemId int16
	Amount byte
	Damage int16 //short
}

func ReadPlaceInPacket(data *[]byte) PlaceInPacket {
	reader := packet.PacketReader{
		Data: data,
	}

	packet := PlaceInPacket{}
	packet.PacketId = reader.ReadPacketId()
	packet.X = reader.ReadInt32()
	packet.Y = reader.ReadByte()
	packet.Z = reader.ReadInt32()
	packet.Face = reader.ReadByte() // Direction
	packet.ItemId = int16(reader.ReadShort())
	packet.Amount = reader.ReadByte()
	packet.Damage = int16(reader.ReadShort())
	return packet
}
