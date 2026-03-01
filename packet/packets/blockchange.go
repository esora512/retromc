package packets

import "github.com/leNicDev/retromc/packet"

type BlockChangeOutPacket struct {
	X         int32
	Y         byte
	Z         int32
	BlockType byte
	BlockMeta byte
}

func (p *BlockChangeOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.BlockChange) // 0x35
	writer.WriteInt32(p.X)
	writer.WriteByte(p.Y)
	writer.WriteInt32(p.Z)
	writer.WriteByte(p.BlockType)
	writer.WriteByte(p.BlockMeta)
	return writer.Bytes()
}
