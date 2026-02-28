package packets

import (
	"github.com/leNicDev/retromc/packet"
)

type HoldingChangeInPacket struct {
	packet.Packet
	Slot int16
}

func ReadHoldingChangeInPacket(data *[]byte) HoldingChangeInPacket {
	reader := packet.PacketReader{
		Data: data,
	}

	packet := HoldingChangeInPacket{}
	packet.PacketId = reader.ReadPacketId()
	packet.Slot = int16(reader.ReadShort())

	//log.Printf("Holding change: %+v", packet)

	return packet
}
