package packets

import (
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet"
)

type PreChunkOutPacket struct {
	X    int32
	Z    int32
	Mode bool // True = 1 (Initialize chunk); False = 0 (Unload chunk)
}

func (p *PreChunkOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.PreChunk)
	writer.WriteInt32(p.X)   // write chunk x position
	writer.WriteInt32(p.Z)   // write chunk z position
	writer.WriteBool(p.Mode) // write pre chunk mode
	return writer.Bytes()
}

type MapChunkOutPacket struct {
	X              int32
	Y              int16
	Z              int32
	SizeX          byte
	SizeY          byte
	SizeZ          byte
	CompressedSize int32
	CompressedData []byte
}

func (p *MapChunkOutPacket) Apply(chunk level.Chunk) {
	p.X = chunk.X
	p.Y = chunk.Y
	p.Z = chunk.Z
	p.SizeX = chunk.SizeX
	p.SizeY = chunk.SizeY
	p.SizeZ = chunk.SizeZ
	p.CompressedData = chunk.CompressData()
	p.CompressedSize = int32(len(p.CompressedData))
}

func (p *MapChunkOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.MapChunk)
	writer.WriteInt32(p.X)              // write chunk x position
	writer.WriteInt16(p.Y)              // write chunk y position
	writer.WriteInt32(p.Z)              // write chunk z position
	writer.WriteByte(p.SizeX)           // write chunk size x
	writer.WriteByte(p.SizeY)           // write chunk size y
	writer.WriteByte(p.SizeZ)           // write chunk size z
	writer.WriteInt32(p.CompressedSize) // write compressed chunk data size
	writer.Write(p.CompressedData)      // write compressed chunk data
	return writer.Bytes()
}
