package level

import (
	"bytes"
	"compress/zlib"
	"math/rand"
	"unsafe"
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
	Logic *ChunkLogic
}

func (c *Chunk) Size() int64 {
	structOverhead := int64(unsafe.Sizeof(*c))
	dataBytes := int64(len(c.Data))
	return structOverhead + dataBytes
}

func (c *Chunk) CompressData() []byte {
	var buf bytes.Buffer
	writer := zlib.NewWriter(&buf)
	writer.Write(c.Data)
	writer.Close()
	return buf.Bytes()
}

const GROUND_LEVEL = 64
const GROUND_DEPTH = 16

// GenerateTemplate fills the chunk: stone below GROUND_LEVEL, air above, dirt on ground layer
// Blocks are stored in XZY order so y = blockIndex % CHUNK_SIZE_Y.
// Nibble arrays pack two 4-bit values per byte: even index → lower nibble (bits 0-3),
// odd index → upper nibble (bits 4-7).
func (c *Chunk) GenerateTemplate() {
	blocksAmount := CHUNK_SIZE_X * CHUNK_SIZE_Y * CHUNK_SIZE_Z
	nibbleCount := blocksAmount / 2

	blockTypes := make([]byte, blocksAmount)
	blockMetadata := make([]byte, nibbleCount)
	blockLight := make([]byte, nibbleCount)
	blockSkyLight := make([]byte, nibbleCount)

	for i := 0; i < blocksAmount; i++ {
		y := i % CHUNK_SIZE_Y
		z := (i / CHUNK_SIZE_Y) % CHUNK_SIZE_Z
		x := i / (CHUNK_SIZE_Y * CHUNK_SIZE_Z)
		isBorder := x == 0 || x == CHUNK_SIZE_X-1 || z == 0 || z == CHUNK_SIZE_Z-1
		//isCenter := x == 7 && z == 7

		var block Block
		if y >= GROUND_LEVEL-GROUND_DEPTH && y < GROUND_LEVEL {
			block = NewStoneBlock()
			if y == GROUND_LEVEL-1 {
				if isBorder {
					block = NewStoneBlock()
				} else {
					block = NewGrassBlock()
				}
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
	c.relightAll()
}

func (c *Chunk) GenerateEmpty() {
	blocksAmount := CHUNK_SIZE_X * CHUNK_SIZE_Y * CHUNK_SIZE_Z
	nibbleCount := blocksAmount / 2

	blockTypes := make([]byte, blocksAmount)
	blockMetadata := make([]byte, nibbleCount)
	blockLight := make([]byte, nibbleCount)
	blockSkyLight := make([]byte, nibbleCount)

	for i := 0; i < blocksAmount; i++ {
		ni := i / 2
		if i%2 == 0 {
			blockSkyLight[ni] = 0x0f
		} else {
			blockSkyLight[ni] |= 0x0f << 4
		}
	}

	c.Data = blockTypes
	c.Data = append(c.Data, blockMetadata...)
	c.Data = append(c.Data, blockLight...)
	c.Data = append(c.Data, blockSkyLight...)
	c.relightAll()
}

// relightColumn recalculates skylight for a single (lx, lz) column using a
// simple top-down scan: skylight is 15 until the first non-transparent
// block is hit (going top to bottom), then 0 for everything below.
// No flood-fill / cave handling — assumes no overhangs reachable from
// underground gaps, which holds for this world generation.
func (c *Chunk) relightColumn(lx, lz int) {
	blocksAmount := CHUNK_SIZE_X * CHUNK_SIZE_Y * CHUNK_SIZE_Z
	nibbleCount := blocksAmount / 2
	skyOffset := blocksAmount + 2*nibbleCount

	lit := true
	for ly := CHUNK_SIZE_Y - 1; ly >= 0; ly-- {
		i := lx*CHUNK_SIZE_Z*CHUNK_SIZE_Y + lz*CHUNK_SIZE_Y + ly
		block := c.GetBlock(lx, ly, lz)

		if lit && !block.IsTransparent() {
			lit = false
		}

		var sky byte
		if lit {
			sky = 0x0f
		} else {
			sky = 0x00
		}

		ni := i / 2
		if i%2 == 0 {
			c.Data[skyOffset+ni] = (c.Data[skyOffset+ni] & 0xf0) | sky
		} else {
			c.Data[skyOffset+ni] = (c.Data[skyOffset+ni] & 0x0f) | (sky << 4)
		}
	}
}

// relightAll relights every column in the chunk.
func (c *Chunk) relightAll() {
	for lx := 0; lx < CHUNK_SIZE_X; lx++ {
		for lz := 0; lz < CHUNK_SIZE_Z; lz++ {
			c.relightColumn(lx, lz)
		}
	}
}

// SetBlock mutates a single block inside an already-generated chunk.
// lx, ly, lz are local (0-based) coordinates within the chunk.
// The Data layout mirrors generate(): blockTypes | blockMetadata | blockLight | blockSkyLight,
// with nibble arrays packed two 4-bit values per byte.
func (c *Chunk) SetBlock(lx, ly, lz int, block Block) {
	blocksAmount := CHUNK_SIZE_X * CHUNK_SIZE_Y * CHUNK_SIZE_Z
	nibbleCount := blocksAmount / 2

	i := lx*CHUNK_SIZE_Z*CHUNK_SIZE_Y + lz*CHUNK_SIZE_Y + ly

	metaOffset := blocksAmount
	lightOffset := blocksAmount + nibbleCount
	skyOffset := blocksAmount + 2*nibbleCount

	c.Data[i] = block.TypeId

	ni := i / 2
	if i%2 == 0 { // lower nibble (bits 0-3)
		c.Data[metaOffset+ni] = (c.Data[metaOffset+ni] & 0xf0) | (block.Metadata & 0x0f)
		c.Data[lightOffset+ni] = (c.Data[lightOffset+ni] & 0xf0) | (block.Light & 0x0f)
		c.Data[skyOffset+ni] = (c.Data[skyOffset+ni] & 0xf0) | (block.SkyLight & 0x0f)
	} else { // upper nibble (bits 4-7)
		c.Data[metaOffset+ni] = (c.Data[metaOffset+ni] & 0x0f) | ((block.Metadata & 0x0f) << 4)
		c.Data[lightOffset+ni] = (c.Data[lightOffset+ni] & 0x0f) | ((block.Light & 0x0f) << 4)
		c.Data[skyOffset+ni] = (c.Data[skyOffset+ni] & 0x0f) | ((block.SkyLight & 0x0f) << 4)
	}
	c.relightColumn(lx, lz)
}

func (c *Chunk) GetBlock(lx, ly, lz int) Block {
	blocksAmount := CHUNK_SIZE_X * CHUNK_SIZE_Y * CHUNK_SIZE_Z
	i := lx*CHUNK_SIZE_Z*CHUNK_SIZE_Y + lz*CHUNK_SIZE_Y + ly

	metaOffset := blocksAmount
	ni := i / 2
	var metadata byte
	if i%2 == 0 {
		metadata = c.Data[metaOffset+ni] & 0x0f
	} else {
		metadata = (c.Data[metaOffset+ni] >> 4) & 0x0f
	}

	return Block{TypeId: c.Data[i], Metadata: metadata}
}

func NewChunk(worldType WorldType) Chunk {
	chunk := Chunk{
		X:     0,
		Y:     0,
		Z:     0,
		SizeX: CHUNK_SIZE_X - 1,
		SizeY: CHUNK_SIZE_Y - 1,
		SizeZ: CHUNK_SIZE_Z - 1,
		Logic: NewChunkLogic(),
	}
	switch worldType {
	case SkyGrid:
		chunk.GenerateSkyGrid()
	default:
		r := rand.Float64()
		if r < 0.95 {
			chunk.GenerateTemplate()
		} else if rand.Float64() < 0.5 {
			chunk.GenerateEmpty()
		} else {
			chunk.GenerateSkyGrid()
		}
	}
	return chunk
}

func mod(a, b int) int {
	r := a % b
	if r < 0 {
		r += b
	}
	return r
}

func (c *Chunk) GenerateSkyGrid() {
	cx, cz := 0, 0
	blocksAmount := CHUNK_SIZE_X * CHUNK_SIZE_Y * CHUNK_SIZE_Z
	nibbleCount := blocksAmount / 2

	blockTypes := make([]byte, blocksAmount)
	blockMetadata := make([]byte, nibbleCount)
	blockLight := make([]byte, nibbleCount)
	blockSkyLight := make([]byte, nibbleCount)

	for i := 0; i < blocksAmount; i++ {
		y := i % CHUNK_SIZE_Y
		z := (i / CHUNK_SIZE_Y) % CHUNK_SIZE_Z
		x := i / (CHUNK_SIZE_Y * CHUNK_SIZE_Z)

		// World coordinates of this block
		worldX := int(cx)*CHUNK_SIZE_X + x
		worldZ := int(cz)*CHUNK_SIZE_Z + z

		// Place a block where world X and Z are multiples of 2,
		// and Y is a multiple of 4 — giving a 3-block vertical gap.
		// The world-space check ensures x=0,z=0 always has a block.
		isSkyGridBlock := (mod(worldX, 3) == 0) && (mod(worldZ, 3) == 0) && (y%4 == 0)
		//isSkyGridBlock := (worldX%2 == 0) && (worldZ%2 == 0) && (y%4 == 0)

		var block Block
		if isSkyGridBlock {
			block = NewRandomBlock()
		} else {
			block = NewAirBlock()
		}
		block.SkyLight = 0x0f

		blockTypes[i] = block.TypeId

		ni := i / 2
		if i%2 == 0 {
			blockMetadata[ni] = block.Metadata & 0x0f
			blockLight[ni] = block.Light & 0x0f
			blockSkyLight[ni] = block.SkyLight & 0x0f
		} else {
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
