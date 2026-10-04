package packets

import (
	"github.com/leNicDev/retromc/packet"
)

type ContainerTransactionPacket struct {
	WindowId     byte
	ActionNumber int16
	Accepted     bool
}

func ReadContainerTransactionPacket(reader *packet.PacketReader) ContainerTransactionPacket {
	return ContainerTransactionPacket{
		WindowId:     reader.ReadByte(),
		ActionNumber: int16(reader.ReadShort()),
		Accepted:     reader.ReadBool(),
	}
}

func (p *ContainerTransactionPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.ContainerTransaction)
	writer.WriteByte(p.WindowId)
	writer.WriteInt16(p.ActionNumber)
	writer.WriteBool(p.Accepted)
	return writer.Bytes()
}
