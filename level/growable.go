package level

import (
	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/packet"
)

const (
	CROP_MAX_STATE = 7
	BLOCK_MAX      = 3
)

type Growable interface {
	Grow(w *World, bk *BlockKey)
	GrowNow(w *World) bool
}

type Wheat struct {
	StartTick int64
	State     byte
}

type Sugarcane struct {
	StartTick int64
}

type Cactus struct {
	StartTick int64
}

type Sapling struct {
	StartTick int64
	WoodType  byte
}

type GrowableDirt struct {
	StartTick int64
}

type PlantRule struct {
	ValidGround  func(groundType byte) bool
	PlantedBlock int16
	UseMeta      bool
}

var PlantRules = map[int16]PlantRule{
	constants.Seeds.Value:         {func(g byte) bool { return g == byte(constants.Farmland.Value) }, constants.Wheat.Value, false},
	constants.Sapling.Value:       {func(g byte) bool { return g == byte(constants.Dirt.Value) || g == byte(constants.Grass.Value) }, constants.Sapling.Value, true},
	constants.SugarcaneItem.Value: {func(g byte) bool { return g == byte(constants.Dirt.Value) || g == byte(constants.Grass.Value) }, constants.Sugarcane.Value, false},
	constants.Cactus.Value:        {func(g byte) bool { return g == byte(constants.Sand.Value) }, constants.Cactus.Value, false},
}

// TODO: Remove duplicate somehow later -_-
type BlockChangeOutPacket struct {
	X         int32
	Y         byte
	Z         int32
	BlockType byte
	BlockMeta byte
}

func (w *World) SetGrowable(block Block, bk BlockKey) {
	chunk := w.GetLoadedChunk(bk.X, bk.Z)
	if chunk == nil {
		return
	}
	if block.IsGrowable() {
		return
	}

	// w.Mu.Lock()
	// defer w.Mu.Unlock()
	logic := chunk.Logic
	if block.TypeId == byte(constants.Wheat.Value) {
		logic.Growables[bk] = &Wheat{StartTick: w.Tick, State: block.Metadata}
	}
	if block.TypeId == byte(constants.Sugarcane.Value) && block.Metadata == 0 {
		logic.Growables[bk] = &Sugarcane{StartTick: w.Tick}
	}
	if block.TypeId == byte(constants.Cactus.Value) && block.Metadata == 0 {
		logic.Growables[bk] = &Cactus{StartTick: w.Tick}
	}
	if block.TypeId == byte(constants.Sapling.Value) {
		logic.Growables[bk] = &Sapling{StartTick: w.Tick, WoodType: block.Metadata}
	}
	if block.TypeId == byte(constants.Dirt.Value) {
		logic.Growables[bk] = &GrowableDirt{StartTick: w.Tick}
	}
}

func (w *World) GrowPhysics() {
	chunks := w.GetRenderedChunks()
	for _, chunk := range chunks {
		logic := chunk.Logic
		for key, growable := range logic.Growables {
			if growable.GrowNow(w) {
				growable.Grow(w, &key)
			}
		}
	}
}

func (p *BlockChangeOutPacket) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.BlockChange)
	writer.WriteInt32(p.X)
	writer.WriteByte(p.Y)
	writer.WriteInt32(p.Z)
	writer.WriteByte(p.BlockType)
	writer.WriteByte(p.BlockMeta)
	return writer.Bytes()
}

func (c *Wheat) GrowNow(w *World) bool {
	diff := w.Tick - c.StartTick
	if diff < 0 {
		c.StartTick = w.Tick
		return false
	}
	if diff > 600 {
		c.StartTick = w.Tick
		return true
	}
	return false
}

func (c *Sugarcane) GrowNow(w *World) bool {
	diff := w.Tick - c.StartTick
	if diff < 0 {
		c.StartTick = w.Tick
		return false
	}
	if diff > 600 {
		c.StartTick = w.Tick
		return true
	}
	return false
}

func (c *Cactus) GrowNow(w *World) bool {
	diff := w.Tick - c.StartTick
	if diff < 0 {
		c.StartTick = w.Tick
		return false
	}
	if diff > 600 {
		c.StartTick = w.Tick
		return true
	}
	return false
}

func (s *Sapling) GrowNow(w *World) bool {
	diff := w.Tick - s.StartTick
	if diff < 0 {
		s.StartTick = w.Tick
		return false
	}
	if diff > 600 {
		s.StartTick = w.Tick
		return true
	}
	return false
}

func (s *GrowableDirt) GrowNow(w *World) bool {
	diff := w.Tick - s.StartTick
	if diff < 0 {
		s.StartTick = w.Tick
		return false
	}
	if diff > 600 {
		s.StartTick = w.Tick
		return true
	}
	return false
}

func (s *GrowableDirt) Grow(w *World, bk *BlockKey) {
	directions := [][2]int{
		{-1, 0}, // west
		{1, 0},  // east
		{0, -1}, // north
		{0, 1},  // south
	}

	connectedToGrass := false
	for _, dir := range directions {
		dx, dz := dir[0], dir[1]
		targetX := bk.X + int32(dx)
		targetZ := bk.Z + int32(dz)
		targetBlock := w.GetBlock(targetX, bk.Y, targetZ)

		if targetBlock.TypeId == byte(constants.Grass.Value) {
			connectedToGrass = true
			break
		}
	}
	if connectedToGrass {
		grass := NewBlockById(constants.Grass.Value, 0)
		w.SetBlock(bk.X, bk.Y, bk.Z, grass)

		blockChange := BlockChangeOutPacket{
			X:         bk.X,
			Y:         bk.Y,
			Z:         bk.Z,
			BlockType: grass.TypeId,
			BlockMeta: grass.Metadata,
		}
		w.BroadcastPacket(blockChange.Serialize())
	}
	cx := WorldToChunkCoord(bk.X)
	cz := WorldToChunkCoord(bk.Z)
	chunk := w.GetOrCreateChunk(cx, cz, w.WorldType)
	logic := chunk.Logic
	delete(logic.Growables, *bk)
}

func (c *Wheat) Grow(w *World, bk *BlockKey) {
	cx := WorldToChunkCoord(bk.X)
	cz := WorldToChunkCoord(bk.Z)
	chunk := w.GetOrCreateChunk(cx, cz, w.WorldType)
	logic := chunk.Logic
	if c.State >= CROP_MAX_STATE {
		delete(logic.Growables, *bk)
		return
	}

	if c.State < CROP_MAX_STATE {
		c.State += 1
	}
	crop := NewBlockById(constants.Wheat.Value, c.State)
	w.SetBlock(bk.X, bk.Y, bk.Z, crop)
	blockChange := BlockChangeOutPacket{
		X:         bk.X,
		Y:         bk.Y,
		Z:         bk.Z,
		BlockType: crop.TypeId,
		BlockMeta: crop.Metadata,
	}
	w.BroadcastPacket(blockChange.Serialize())
}

func (c *Sugarcane) Grow(w *World, bk *BlockKey) {
	baseY := int(bk.Y)
	height := 0
	for {
		b := w.GetBlock(bk.X, byte(baseY+height), bk.Z)
		if b.IsAir() {
			break
		}
		height++
	}

	if height >= BLOCK_MAX {
		return
	}

	cane := NewBlockById(constants.Sugarcane.Value, 1)
	w.SetBlock(bk.X, byte(baseY+height), bk.Z, cane)

	blockChange := BlockChangeOutPacket{
		X:         bk.X,
		Y:         byte(baseY + height),
		Z:         bk.Z,
		BlockType: cane.TypeId,
		BlockMeta: cane.Metadata,
	}
	w.BroadcastPacket(blockChange.Serialize())
}

func (c *Cactus) Grow(w *World, bk *BlockKey) {
	baseY := int(bk.Y)
	height := 0
	for {
		b := w.GetBlock(bk.X, byte(baseY+height), bk.Z)
		if b.IsAir() {
			break
		}
		height++
	}

	if height >= BLOCK_MAX {
		return
	}

	cactus := NewBlockById(constants.Cactus.Value, 1)
	w.SetBlock(bk.X, byte(baseY+height), bk.Z, cactus)

	blockChange := BlockChangeOutPacket{
		X:         bk.X,
		Y:         byte(baseY + height),
		Z:         bk.Z,
		BlockType: cactus.TypeId,
		BlockMeta: cactus.Metadata,
	}
	w.BroadcastPacket(blockChange.Serialize())
}

func (s *Sapling) Grow(w *World, bk *BlockKey) {
	log := NewBlockById(constants.Log.Value, s.WoodType)
	trunkHeight := 5
	for i := 0; i < trunkHeight; i++ {
		w.SetBlock(bk.X, bk.Y+byte(i), bk.Z, log)
		blockChange := BlockChangeOutPacket{
			X:         bk.X,
			Y:         bk.Y + byte(i),
			Z:         bk.Z,
			BlockType: log.TypeId,
			BlockMeta: log.Metadata,
		}
		w.BroadcastPacket(blockChange.Serialize())
	}

	leaves := NewBlockById(constants.Leaves.Value, s.WoodType)
	topY := bk.Y + byte(trunkHeight-3) // one above the last log

	// layer offsets: [dy] = list of (dx, dz) to place
	leafLayers := [4][][2]int{
		// dy=0 and dy=1: 5x5 with corners cut
		{
			{-2, -1}, {-2, 0}, {-2, 1},
			{-1, -2}, {-1, -1}, {-1, 0}, {-1, 1}, {-1, 2},
			{0, -2}, {0, -1}, {0, 1}, {0, 2}, // skip trunk {0,0}
			{1, -2}, {1, -1}, {1, 0}, {1, 1}, {1, 2},
			{2, -1}, {2, 0}, {2, 1},
		},
		// dy=1: same as dy=0 but include trunk column too
		{
			{-2, -1}, {-2, 0}, {-2, 1},
			{-1, -2}, {-1, -1}, {-1, 0}, {-1, 1}, {-1, 2},
			{0, -2}, {0, -1}, {0, 1}, {0, 2},
			{1, -2}, {1, -1}, {1, 0}, {1, 1}, {1, 2},
			{2, -1}, {2, 0}, {2, 1},
		},
		// dy=2: 3x3 full
		{
			{-1, -1}, {-1, 0}, {-1, 1},
			{0, -1}, {0, 1},
			{1, -1}, {1, 0}, {1, 1},
		},
		// dy=3: plus/cross cap
		{
			{0, 0},
			{0, -1}, {0, 1},
			{-1, 0}, {1, 0},
		},
	}

	for dy, offsets := range leafLayers {
		for _, off := range offsets {
			dx, dz := off[0], off[1]
			w.SetBlock(bk.X+int32(dx), topY+byte(dy), bk.Z+int32(dz), leaves)
			blockChange := BlockChangeOutPacket{
				X:         bk.X + int32(dx),
				Y:         topY + byte(dy),
				Z:         bk.Z + int32(dz),
				BlockType: leaves.TypeId,
				BlockMeta: leaves.Metadata,
			}
			w.BroadcastPacket(blockChange.Serialize())
		}
	}
	cx := WorldToChunkCoord(bk.X)
	cz := WorldToChunkCoord(bk.Z)
	chunk := w.GetOrCreateChunk(cx, cz, w.WorldType)
	logic := chunk.Logic
	delete(logic.Growables, *bk)
}

func PlantGrowable(w *World, typeId int16, x int32, y byte, z int32, meta byte) *Block {
	growable := NewBlockById(typeId, meta)
	w.SetBlock(x, y, z, growable)
	return &growable
}
