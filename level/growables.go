package level

import (
	"math/rand"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/player"
)

const (
	CROP_MAX_STATE    = 7
	BLOCK_MAX         = 3
	MinTickY          = 60
	MaxTickY          = 80
	ScanRadius        = 16
	GrowChancePerScan = 10
)

type GrowHandler func(w *World, x int32, y byte, z int32, block constants.WBlock, dim int32)

var GrowHandlers = map[byte]GrowHandler{
	byte(constants.Wheat.Value):     growWheat,
	byte(constants.Sugarcane.Value): growSugarcane,
	byte(constants.Cactus.Value):    growCactus,
	byte(constants.Sapling.Value):   growSapling,
	byte(constants.Dirt.Value):      growDirtToGrass,
}

func growWheat(w *World, x int32, y byte, z int32, block constants.WBlock, dim int32) {
	if block.Metadata >= CROP_MAX_STATE {
		return
	}
	if rand.Intn(3) != 0 {
		return
	}
	crop := constants.NewBlockById(constants.Wheat.Value, block.Metadata+1)
	w.SetBlock(x, y, z, crop, dim)
	w.BroadcastBlockChange(x, int32(y), z, crop.TypeId, crop.Metadata)
}

func growSapling(w *World, x int32, y byte, z int32, block constants.WBlock, dim int32) {
	if rand.Intn(20) != 0 {
		return
	}
	buildTree(w, x, y, z, block.Metadata, dim)
}

func growSugarcane(w *World, x int32, y byte, z int32, block constants.WBlock, dim int32) {
	if rand.Intn(4) != 0 {
		return
	}
	if w.GetBlock(x, y+1, z, dim).TypeId == byte(constants.Sugarcane.Value) {
		return
	}

	height := byte(1)
	for w.GetBlock(x, y-height, z, dim).TypeId == byte(constants.Sugarcane.Value) {
		height++
	}
	if int(height) >= BLOCK_MAX {
		return
	}

	cane := constants.NewBlockById(constants.Sugarcane.Value, 0)
	w.SetBlock(x, y+1, z, cane, dim)
	w.BroadcastBlockChange(x, int32(y)+1, z, cane.TypeId, cane.Metadata)
}

func growCactus(w *World, x int32, y byte, z int32, block constants.WBlock, dim int32) {
	if rand.Intn(4) != 0 {
		return
	}
	if w.GetBlock(x, y+1, z, dim).TypeId == byte(constants.Cactus.Value) {
		return
	}

	height := byte(1)
	for w.GetBlock(x, y-height, z, dim).TypeId == byte(constants.Cactus.Value) {
		height++
	}
	if int(height) >= BLOCK_MAX {
		return
	}

	cactus := constants.NewBlockById(constants.Cactus.Value, 0)
	w.SetBlock(x, y+1, z, cactus, dim)
	w.BroadcastBlockChange(x, int32(y)+1, z, cactus.TypeId, cactus.Metadata)
}

func growDirtToGrass(w *World, x int32, y byte, z int32, block constants.WBlock, dim int32) {
	if rand.Intn(10) != 0 {
		return
	}
	for _, d := range [][2]int32{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		if w.GetBlock(x+d[0], y, z+d[1], dim).TypeId == byte(constants.Grass.Value) {
			grass := constants.NewBlockById(constants.Grass.Value, 0)
			w.SetBlock(x, y, z, grass, dim)
			w.BroadcastBlockChange(x, int32(y), z, grass.TypeId, grass.Metadata)
			return
		}
	}
}

func buildTree(w *World, x int32, y byte, z int32, woodType byte, dim int32) {
	log := constants.NewBlockById(constants.Log.Value, woodType)
	trunkHeight := 5
	for i := 0; i < trunkHeight; i++ {
		w.SetBlock(x, y+byte(i), z, log, dim)
		w.BroadcastBlockChange(x, int32(y)+int32(i), z, log.TypeId, log.Metadata)
	}

	leaves := constants.NewBlockById(constants.Leaves.Value, woodType)
	topY := y + byte(trunkHeight-3) // one above the last log

	// layer offsets: [dy] = list of (dx, dz) to place
	leafLayers := [4][][2]int{
		// dy=0: 5x5 with corners cut
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
			w.SetBlock(x+int32(dx), topY+byte(dy), z+int32(dz), leaves, dim)
			w.BroadcastBlockChange(x+int32(dx), int32(topY)+int32(dy), z+int32(dz), leaves.TypeId, leaves.Metadata)
		}
	}
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

func PlantGrowable(w *World, typeId int16, x int32, y byte, z int32, meta byte, dim int32) *constants.WBlock {
	growable := constants.NewBlockById(typeId, meta)
	w.SetBlock(x, y, z, growable, dim)
	return &growable
}

const (
	TickYPad        = 2 // vertical window around the player
	WorldMinY       = 0
	WorldMaxY       = 127
	RandomTickSpeed = 24
)

func (w *World) RandomTickPhysics() {
	for _, dim := range []int32{0, -1} {
		w.randomScan(dim)
	}
}

func playerYWindow(playerY float64) (byte, byte) {
	yMin := int(playerY) - TickYPad
	yMax := int(playerY) + TickYPad
	if yMin < WorldMinY {
		yMin = WorldMinY
	}
	if yMax > WorldMaxY {
		yMax = WorldMaxY
	}
	if yMax <= yMin {
		yMax = yMin + 1
	}
	return byte(yMin), byte(yMax)
}

func (w *World) randomTickChunk(chunk *Chunk, dim int32, yMin, yMax byte) {
	baseX := chunk.X * 16
	baseZ := chunk.Z * 16
	yRange := int(yMax - yMin)

	for i := 0; i < RandomTickSpeed; i++ {
		x := baseX + int32(rand.Intn(16))
		z := baseZ + int32(rand.Intn(16))
		y := yMin + byte(rand.Intn(yRange))

		block := w.GetBlock(x, y, z, dim)
		handler, ok := GrowHandlers[block.TypeId]
		if !ok {
			continue
		}
		handler(w, x, y, z, block, dim)
	}
}

func (w *World) randomScan(dim int32) {
	w.Mu.RLock()
	var players []*player.Player
	for _, pl := range w.Players {
		if pl.Dimension == dim {
			players = append(players, pl)
		}
	}
	w.Mu.RUnlock()

	for _, pl := range players {
		yMin, yMax := playerYWindow(pl.Y)
		px := int32(pl.X)
		pz := int32(pl.Z)

		for x := px - ScanRadius; x <= px+ScanRadius; x++ {
			for z := pz - ScanRadius; z <= pz+ScanRadius; z++ {
				for y := yMin; y < yMax; y++ {
					block := w.GetBlock(x, y, z, dim)
					handler, ok := GrowHandlers[block.TypeId]
					if !ok {
						continue
					}
					if rand.Intn(GrowChancePerScan) != 0 {
						continue
					}
					handler(w, x, y, z, block, dim)
				}
			}
		}
	}
}
