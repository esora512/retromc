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

const (
	StateBeginRain byte = 1
	StateEndRain   byte = 2
)

type NewStatePacket struct {
	Reason byte
}

func (p *NewStatePacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.NewState)
	w.WriteByte(p.Reason)
	return w.Bytes()
}

type ThunderboltPacket struct {
	EntityId int32
	X, Y, Z  int32
}

func (p *ThunderboltPacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.Thunderbolt)
	w.WriteInt32(p.EntityId)
	w.WriteBool(true)
	w.WriteInt32(p.X * 32)
	w.WriteInt32(p.Y * 32)
	w.WriteInt32(p.Z * 32)
	return w.Bytes()
}
