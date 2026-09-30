package packets

import (
	"bytes"
	"encoding/binary"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet"
)

type SetChunkVisibilityPacket struct {
	X    int32
	Z    int32
	Mode bool // True = 1 (Initialize chunk); False = 0 (Unload chunk)
}

func (p *SetChunkVisibilityPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.SetChunkVisibility)
	writer.WriteInt32(p.X)   // write chunk x position
	writer.WriteInt32(p.Z)   // write chunk z position
	writer.WriteBool(p.Mode) // write pre chunk mode
	return writer.Bytes()
}

// NewChunkBlockRegionPacket serializes a full chunk, compressing its data straight into the packet buffer.
func NewChunkBlockRegionPacket(chunk *level.Chunk, light *level.ChunkLight) []byte {
	var buf bytes.Buffer
	buf.Grow(16 * 1024)
	buf.WriteByte(packet.ChunkBlockRegion)
	binary.Write(&buf, binary.BigEndian, chunk.X)
	binary.Write(&buf, binary.BigEndian, chunk.Y)
	binary.Write(&buf, binary.BigEndian, chunk.Z)
	buf.Write([]byte{chunk.SizeX, chunk.SizeY, chunk.SizeZ})
	sizeOffset := buf.Len()
	buf.Write(make([]byte, 4)) // compressed size, filled in below
	chunk.WriteCompressed(&buf, light)
	out := buf.Bytes()
	binary.BigEndian.PutUint32(out[sizeOffset:], uint32(len(out)-sizeOffset-4))
	return out
}

type WorldEventPacket struct {
	EffectId int32
	X        int32
	Y        byte
	Z        int32
	Data     int32
}

func (p *WorldEventPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.WorldEvent)
	writer.WriteInt32(p.EffectId)
	writer.WriteInt32(p.X)
	writer.WriteByte(p.Y)
	writer.WriteInt32(p.Z)
	writer.WriteInt32(p.Data)
	return writer.Bytes()
}
