package level

import (
	"bytes"
	"compress/zlib"
)

const (
	CHUNK_SIZE_X = 16
	CHUNK_SIZE_Y = 128
	CHUNK_SIZE_Z = 16
)

type Chunk struct {
	X     int32
	Y     int16
	Z     int32
	SizeX byte
	SizeY byte
	SizeZ byte
	Data  []byte
}

// compress chunk data using zlib
func (c *Chunk) CompressData() []byte {
	var buf bytes.Buffer
	writer := zlib.NewWriter(&buf)
	writer.Write(c.Data)
	writer.Close()
	return buf.Bytes()
}

const GROUND_LEVEL = 64

// generate fills the chunk: stone below GROUND_LEVEL, air above.
// Blocks are stored in XZY order so y = blockIndex % CHUNK_SIZE_Y.
// Nibble arrays pack two 4-bit values per byte: even index → lower nibble (bits 0-3),
// odd index → upper nibble (bits 4-7).
func (c *Chunk) generate() {
	blocksAmount := CHUNK_SIZE_X * CHUNK_SIZE_Y * CHUNK_SIZE_Z
	nibbleCount := blocksAmount / 2

	blockTypes := make([]byte, blocksAmount)
	blockMetadata := make([]byte, nibbleCount)
	blockLight := make([]byte, nibbleCount)
	blockSkyLight := make([]byte, nibbleCount)

	for i := 0; i < blocksAmount; i++ {
		y := i % CHUNK_SIZE_Y

		var block Block
		if y < GROUND_LEVEL {
			block = NewStoneBlock()
			// The top stone layer is exposed to sky, so it must receive full skylight.
			// Without this, the surface renders pitch black (skylight=0, blocklight=0).
			if y == GROUND_LEVEL-1 {
				block.SkyLight = 0x0f
			}
		} else {
			block = NewAirBlock()
		}

		blockTypes[i] = block.TypeId

		ni := i / 2
		if i%2 == 0 { // lower nibble (bits 0-3)
			blockMetadata[ni] = block.Metadata & 0x0f
			blockLight[ni] = block.Light & 0x0f
			blockSkyLight[ni] = block.SkyLight & 0x0f
		} else { // upper nibble (bits 4-7)
			blockMetadata[ni] |= (block.Metadata & 0x0f) << 4
			blockLight[ni] |= (block.Light & 0x0f) << 4
			blockSkyLight[ni] |= (block.SkyLight & 0x0f) << 4
		}
	}

	c.Data = blockTypes
	c.Data = append(c.Data, blockMetadata...)
	c.Data = append(c.Data, blockLight...)
	c.Data = append(c.Data, blockSkyLight...)
}

func NewChunk() Chunk {
	chunk := Chunk{
		X:     0,
		Y:     0,
		Z:     0,
		SizeX: CHUNK_SIZE_X - 1,
		SizeY: CHUNK_SIZE_Y - 1,
		SizeZ: CHUNK_SIZE_Z - 1,
	}
	chunk.generate()
	return chunk
}
