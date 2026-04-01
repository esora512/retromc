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
	writer.WriteByte(packet.Transaction) 
	writer.WriteByte(p.WindowId)        
	writer.WriteInt16(p.ActionNumber)   
	writer.WriteBool(p.Accepted)
	return writer.Bytes()
}
