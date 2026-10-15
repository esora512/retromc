package packets

import (
	"bytes"
	"encoding/binary"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/player"
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
func NewChunkBlockRegionPacket(chunk *level.Chunk) []byte {
	return NewChunkRegionPacket(chunk, 0, 0, 0, int(chunk.SizeX), int(chunk.SizeY), int(chunk.SizeZ))
}

// NewChunkRegionPacket serializes a box of a chunk; bounds are local and inclusive, y0 even and y1 odd.
func NewChunkRegionPacket(chunk *level.Chunk, x0, y0, z0, x1, y1, z1 int) []byte {
	var buf bytes.Buffer
	buf.Grow(16 * 1024)
	buf.WriteByte(packet.ChunkBlockRegion)
	binary.Write(&buf, binary.BigEndian, chunk.X+int32(x0))
	binary.Write(&buf, binary.BigEndian, chunk.Y+int16(y0))
	binary.Write(&buf, binary.BigEndian, chunk.Z+int32(z0))
	buf.Write([]byte{byte(x1 - x0), byte(y1 - y0), byte(z1 - z0)})
	sizeOffset := buf.Len()
	buf.Write(make([]byte, 4)) // compressed size, filled in below
	chunk.WriteRegionCompressed(&buf, x0, y0, z0, x1, y1, z1)
	out := buf.Bytes()
	binary.BigEndian.PutUint32(out[sizeOffset:], uint32(len(out)-sizeOffset-4))
	return out
}

// BroadcastChunkRegion sends a box of a chunk to every player that already has the chunk.
func BroadcastChunkRegion(w *level.World, chunk *level.Chunk, dim int32, x0, y0, z0, x1, y1, z1 int) {
	cx, cz := chunk.X>>4, chunk.Z>>4
	var data []byte
	w.ForEachPlayer(func(pl *player.Player) {
		if pl.Dimension != dim {
			return
		}
		pl.SentChunksMu.Lock()
		has := pl.SentChunks.Has(cx, cz)
		pl.SentChunksMu.Unlock()
		if !has {
			return
		}
		if data == nil {
			data = NewChunkRegionPacket(chunk, x0, y0, z0, x1, y1, z1)
		}
		pl.Connection.Write(data)
	})
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
