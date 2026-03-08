package packets

import (
	"github.com/leNicDev/retromc/packet"
)

type KeepAliveInPacket struct {
	packet.Packet
}

func ReadKeepAliveInPacket(reader *packet.PacketReader) KeepAliveInPacket {
	p := KeepAliveInPacket{}
	p.PacketId = reader.GetPacketId()
	return p
}

type KeepAliveOutPacket struct {
	packet.Packet
}

func (p *KeepAliveOutPacket) Serialize() []byte {
	return []byte{packet.KeepAlive}
}
