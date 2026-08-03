package packets

import "github.com/leNicDev/retromc/packet"

type SetTimePacket struct {
	Time int64
}

func (p *SetTimePacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.SetTime)
	w.WriteInt64(p.Time)
	return w.Bytes()
}
