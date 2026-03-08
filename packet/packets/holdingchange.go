package packets

import (
	"github.com/leNicDev/retromc/packet"
)

type HoldingChangeInPacket struct {
	packet.Packet
	Slot int16
}

func ReadHoldingChangeInPacket(reader packet.PacketReader) HoldingChangeInPacket {
	packet := HoldingChangeInPacket{}
	packet.PacketId = reader.GetPacketId()
	packet.Slot = int16(reader.ReadShort())

	//log.Printf("Holding change: %+v", packet)

	return packet
}
