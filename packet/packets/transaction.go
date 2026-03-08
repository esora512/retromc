package packets

import (
	"github.com/leNicDev/retromc/packet"
)

type TransactionOutPacket struct {
	WindowId     byte
	ActionNumber int16
	Accepted     bool
}

func (p *TransactionOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.Transaction) // write packet id
	writer.WriteByte(p.WindowId)         // write window id
	writer.WriteInt16(p.ActionNumber)    // write action number
	writer.WriteBool(p.Accepted)         // write accepted
	return writer.Bytes()
}
