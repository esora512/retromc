package level

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"unsafe"

	"github.com/leNicDev/retromc/mcregion"
)

const (
	CHUNK_SIZE_X = 16
	CHUNK_SIZE_Y = 128
	CHUNK_SIZE_Z = 16
)

type Chunk struct {
	X          int32
	Y          int16
	Z          int32
	SizeX      byte
	SizeY      byte
	SizeZ      byte
	Data       []byte
	Logic      *ChunkLogic
	HasChanged bool
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
	c.HasChanged = false
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
	c.HasChanged = false

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

func NewChunk() Chunk {
	chunk := Chunk{
		X:     0,
		Y:     0,
		Z:     0,
		SizeX: CHUNK_SIZE_X - 1,
		SizeY: CHUNK_SIZE_Y - 1,
		SizeZ: CHUNK_SIZE_Z - 1,
		Logic: NewChunkLogic(),
	}
	chunk.GenerateEmpty()
	chunk.HasChanged = false
	return chunk
}

func chunkRand(worldSeed int64, cx, cz int32) *rand.Rand {
	h := uint64(worldSeed)
	h ^= uint64(uint32(cx)) * 0x9E3779B97F4A7C15
	h ^= uint64(uint32(cz)) * 0xC2B2AE3D27D4EB4F
	h ^= h >> 33
	h *= 0xFF51AFD7ED558CCD
	h ^= h >> 33
	h *= 0xC4CEB9FE1A85EC53
	h ^= h >> 33

	return rand.New(rand.NewSource(int64(h)))
}

func (w *World) generateChunk(cx, cz int32, worldType WorldType) *Chunk {
	worldX := cx * CHUNK_SIZE_X
	worldZ := cz * CHUNK_SIZE_Z

	chunk := &Chunk{
		X:     worldX,
		Y:     0,
		Z:     worldZ,
		SizeX: CHUNK_SIZE_X - 1,
		SizeY: CHUNK_SIZE_Y - 1,
		SizeZ: CHUNK_SIZE_Z - 1,
		Logic: NewChunkLogic(),
	}

	switch worldType {
	case SkyGrid:
		chunk.GenerateSkyGrid()
	case Template:
		chunk.GenerateTemplate()
	case Esorian:
		r := chunkRand(w.Seed, cx, cz)
		if r.Float64() < 0.95 {
			chunk.GenerateTemplate()
		} else if r.Float64() < 0.5 {
			chunk.GenerateEmpty()
		} else {
			chunk.GenerateSkyGrid()
		}
	case Maze:
		chunk.GenerateMaze(w.Seed, cx, cz)
	case Default:
		chunk.GenerateBareironBiomes(uint32(w.Seed), cx, cz)

	default:
		log.Println("Entering wrong branch, defaulting to default")
		chunk.GenerateBareironBiomes(uint32(w.Seed), cx, cz)
	}
	chunk.HasChanged = false
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

// Global Chunk Operations
type WorldType int

const (
	Template WorldType = iota
	Empty
	SkyGrid
	Default
	Esorian
	Maze
)

type DroppedItem struct {
	EntityId    int32
	ItemId      int32
	Amount      byte
	Metadata    byte
	X, Y, Z     int32
	PickupDelay int32
	Dim         int32

	VelX, VelY, VelZ float64
}

func (d *DroppedItem) GetEntityId() int32 {
	return d.EntityId
}

func (d *DroppedItem) GetHP() int16 {
	return 20
}

func (d *DroppedItem) SetHP(hp int16) {}

func (d *DroppedItem) GetName() string {
	return fmt.Sprintf("Entity %d", d.EntityId)
}

func (d *DroppedItem) GetPosition() (float64, float64, float64) {
	return float64(d.X), float64(d.Y), float64(d.Z)
}

func (d *DroppedItem) SetPosition(x, y, z float64) {}

func (d *DroppedItem) IsRideable() bool { return false }

func (d *DroppedItem) IsPlayer() bool { return false }

func (d *DroppedItem) IsMob() bool { return false }

func (d *DroppedItem) GetLoggedIn() bool { return false }

func (d *DroppedItem) GetDim() int32 { return d.Dim }

func (d *DroppedItem) GetVelocity() (float64, float64, float64) {return d.VelX, d.VelY, d.VelZ}

func (d *DroppedItem) IsItem() bool {return  true}

type ChunkLogic struct {
	Growables    map[BlockKey]Growable
	DroppedItems map[int32]*DroppedItem
}

func (w *World) chunksFor(dim int32) map[ChunkCoord]*Chunk {
	if dim == -1 {
		return w.nChunks
	}
	return w.oChunks
}

func NewChunkLogic() *ChunkLogic {
	return &ChunkLogic{
		Growables:    make(map[BlockKey]Growable),
		DroppedItems: make(map[int32]*DroppedItem),
	}
}

func (w *World) GetLoadedChunk(x, z, dim int32) *Chunk {
	cx := WorldToChunkCoord(x)
	cz := WorldToChunkCoord(z)
	return w.chunksFor(dim)[ChunkCoord{cx, cz}]
}

func (w *World) wantedChunks(dim int32) map[ChunkCoord]struct{} {
	wanted := make(map[ChunkCoord]struct{})
	for _, pl := range w.Players {
		if pl.Dimension != dim {
			continue
		}
		cx := WorldToChunkCoord(int32(pl.X))
		cz := WorldToChunkCoord(int32(pl.Z))
		for dx := -VIEW_DISTANCE; dx <= VIEW_DISTANCE; dx++ {
			for dz := -VIEW_DISTANCE; dz <= VIEW_DISTANCE; dz++ {
				wanted[ChunkCoord{X: cx + int32(dx), Z: cz + int32(dz)}] = struct{}{}
			}
		}
	}
	return wanted
}

func (w *World) GetRenderedChunks(dim int32) []*Chunk {
	wanted := w.wantedChunks(dim)
	w.Mu.RLock()
	defer w.Mu.RUnlock()
	src := w.chunksFor(dim)
	chunks := make([]*Chunk, 0, len(wanted))
	for wa := range wanted {
		if c, ok := src[wa]; ok {
			chunks = append(chunks, c)
		}
	}
	return chunks
}

func (w *World) PlayerActiveChunks(radius, dim int32) []*Chunk {
	seen := make(map[ChunkCoord]struct{})
	chunks := make([]*Chunk, 0, len(w.Players)*9)
	for _, pl := range w.Players {
		if pl.Dimension != dim {
			continue
		}
		cx := WorldToChunkCoord(int32(pl.X))
		cz := WorldToChunkCoord(int32(pl.Z))
		for dx := -radius; dx <= radius; dx++ {
			for dz := -radius; dz <= radius; dz++ {
				coord := ChunkCoord{X: cx + dx, Z: cz + dz}
				if _, dup := seen[coord]; dup {
					continue
				}
				seen[coord] = struct{}{}
				src := w.chunksFor(dim)
				if c, ok := src[coord]; ok {
					chunks = append(chunks, c)
				}
			}
		}
	}
	return chunks
}

func (w *World) PopUnusedChunks(dim int32) map[ChunkCoord]*Chunk {
	wanted := w.wantedChunks(dim)

	w.Mu.Lock()
	defer w.Mu.Unlock()

	src := w.chunksFor(dim)
	var removed map[ChunkCoord]*Chunk
	for coord, ch := range src {
		if _, ok := wanted[coord]; !ok {
			if removed == nil {
				removed = make(map[ChunkCoord]*Chunk, 4)
			}
			removed[coord] = ch
			delete(src, coord)
		}
	}
	if len(removed) > 0 {
		//log.Printf("Popping %d chunks (dim %d)", len(removed), dim)
	}
	return removed
}

func (w *World) IsLoaded(x, z, dim int32) bool {
	cx := WorldToChunkCoord(x)
	cz := WorldToChunkCoord(z)
	_, ok := w.chunksFor(dim)[ChunkCoord{cx, cz}]
	return ok
}

func (w *World) Size() int64 {
	w.Mu.RLock()
	defer w.Mu.RUnlock()
	var total int64
	for _, c := range w.oChunks {
		if c == nil {
			continue
		}
		total += c.Size()
	}
	for _, c := range w.nChunks {
		if c == nil {
			continue
		}
		total += c.Size()
	}
	return total
}

func (c *Chunk) SizeString() string {
	return formatBytes(c.Size())
}

func (w *World) SizeString() string {
	return formatBytes(w.Size())
}

func formatBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func (w *World) LoadChunks(dim int32) []*Chunk {
	w.Mu.RLock()
	defer w.Mu.RUnlock()
	src := w.chunksFor(dim)
	chunks := make([]*Chunk, 0, len(src))
	for _, c := range src {
		chunks = append(chunks, c)
	}
	return chunks
}

func (w *World) ChunkExists(cx, cz, dim int32) bool {
	_, ok := w.chunksFor(dim)[ChunkCoord{cx, cz}]
	return ok
}

func (w *World) GetOrCreateChunk(cx, cz, dim int32) *Chunk {
	key := ChunkCoord{cx, cz}
	chunks := w.chunksFor(dim)

	w.Mu.RLock()
	ch, ok := chunks[key]
	w.Mu.RUnlock()
	if ok {
		return ch
	}

	sfKey := fmt.Sprintf("%d|%d|%d", dim, cx, cz)
	v, err, _ := w.chunkLoadGroup.Do(sfKey, func() (interface{}, error) {
		w.Mu.RLock()
		if ch, ok := chunks[key]; ok {
			w.Mu.RUnlock()
			return ch, nil
		}
		w.Mu.RUnlock()

		c := w.loadOrGenerateChunkFromDiskOrGen(cx, cz, dim)

		w.Mu.Lock()
		chunks[key] = c
		w.Mu.Unlock()

		return c, nil
	})
	if err != nil {
		return nil
	}
	return v.(*Chunk)
}

func (w *World) loadOrGenerateChunkFromDiskOrGen(cx, cz, dim int32) *Chunk {
	if w.WorldDir != "" {
		dir := w.WorldDir
		if dim == -1 {
			dir = filepath.Join(w.WorldDir, "DIM-1")
		}
		rx, rz := cx>>5, cz>>5
		lx, lz := cx&31, cz&31
		regionPath := filepath.Join(dir, "region", mcregion.RegionFileName(rx*32, rz*32))

		lvl, err := mcregion.ReadChunk(regionPath, lx, lz)
		if err != nil {
			log.Printf("chunk (%d,%d) dim %d: read failed, regenerating: %v", cx, cz, dim, err)
		} else if lvl != nil {
			c, err := w.readChunkFromNBT(lvl, cx, cz)
			if err != nil {
				log.Printf("chunk (%d,%d) dim %d: decode failed, regenerating: %v", cx, cz, dim, err)
			} else {
				return c
			}
		}
	}

	worldType := w.WorldType
	if dim == -1 {
		worldType = Maze
	}
	c := w.generateChunk(cx, cz, worldType)
	c.X = cx * CHUNK_SIZE_X
	c.Z = cz * CHUNK_SIZE_Z
	return c
}

func (w *World) NewEmptyChunk(cx, cz int32) *Chunk {
	c := w.generateChunk(cx, cz, Empty)
	c.X = cx * CHUNK_SIZE_X
	c.Z = cz * CHUNK_SIZE_Z
	return c
}
