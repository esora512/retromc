package packets

import (
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/player"
)

type CloseWindowInPacket struct {
	packet.Packet
	WindowId byte
}

func ReadCloseWindowInPacket(reader *packet.PacketReader, pl *player.Player) CloseWindowInPacket {
	p := CloseWindowInPacket{}
	p.PacketId = reader.GetPacketId()
	p.WindowId = reader.ReadByte()
	if p.WindowId == 1 {
		pl.Workbench.ClearGrid()
	}
	return p
}
