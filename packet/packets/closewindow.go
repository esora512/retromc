package packets

import "github.com/leNicDev/retromc/packet"

type CloseWindowInPacket struct {
	packet.Packet
	WindowId byte
}

func ReadCloseWindowInPacket(reader *packet.PacketReader) CloseWindowInPacket {
	p := CloseWindowInPacket{}
	p.PacketId = reader.GetPacketId()
	p.WindowId = reader.ReadByte()
	return p
}
