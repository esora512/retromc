package level

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"time"

	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/mcregion"
)

const CHUNK_HEIGHT = 128

// buildItemNBT encodes one inventory slot. Empty slots should just be
// omitted from the Items list entirely — Beta format doesn't pad it.
func buildItemNBT(slot int, itemID int16, damage int16, count byte) *mcregion.Compound {
	item := mcregion.NewCompound()
	item.Short("id", itemID)
	item.Short("Damage", damage)
	item.Byte("Count", count)
	item.Byte("Slot", byte(slot))
	return item
}

func buildChestNBT(x, y, z int32, chest *inventory.Chest) *mcregion.Compound {
	var items []*mcregion.Compound
	for slot, stack := range chest.Items {
		if stack.TypeId == -1 || stack.Count == 0 {
			continue
		}
		items = append(items, buildItemNBT(slot, stack.TypeId, int16(stack.Metadata), stack.Count))
	}

	comp := mcregion.NewCompound()
	comp.String("id", "Chest")
	comp.Int("x", x)
	comp.Int("y", y)
	comp.Int("z", z)
	comp.CompoundList("Items", items)
	return comp
}

func buildFurnaceNBT(x, y, z int32, furnace *inventory.Furnace) *mcregion.Compound {
	var items []*mcregion.Compound
	if furnace.Items[0].TypeId != -1 && furnace.Items[0].Count > 0 {
		items = append(items, buildItemNBT(0, furnace.Items[0].TypeId, int16(furnace.Items[0].Metadata), furnace.Items[0].Count))
	}
	if furnace.Items[1].TypeId != -1 && furnace.Items[1].Count > 0 {
		items = append(items, buildItemNBT(1, furnace.Items[1].TypeId, int16(furnace.Items[1].Metadata), furnace.Items[1].Count))
	}
	if furnace.Items[2].TypeId != -1 && furnace.Items[2].Count > 0 {
		items = append(items, buildItemNBT(2, furnace.Items[2].TypeId, int16(furnace.Items[2].Metadata), furnace.Items[2].Count))
	}

	comp := mcregion.NewCompound()
	comp.String("id", "Furnace")
	comp.Int("x", x)
	comp.Int("y", y)
	comp.Int("z", z)
	comp.CompoundList("Items", items)
	return comp
}

func buildDispenserNBT(x, y, z int32, dispenser *inventory.Dispenser) *mcregion.Compound {
	var items []*mcregion.Compound
	for slot, stack := range dispenser.Items {
		if stack.TypeId == -1 || stack.Count == 0 {
			continue
		}
		items = append(items, buildItemNBT(slot, stack.TypeId, int16(stack.Metadata), stack.Count))
	}

	comp := mcregion.NewCompound()
	comp.String("id", "Dispenser")
	comp.Int("x", x)
	comp.Int("y", y)
	comp.Int("z", z)
	comp.CompoundList("Items", items)
	return comp
}

// Uses chunk.GetBlock to build NBT chunk block by block
func (w *World) buildChunkNBT(ch *Chunk, cx, cz int32, tick int64) *mcregion.Compound {
	blocks := make([]byte, 16*CHUNK_HEIGHT*16)
	data := make([]byte, len(blocks)/2)
	skyLight := make([]byte, len(blocks)/2)
	blockLight := make([]byte, len(blocks)/2)
	heightMap := make([]byte, 256)

	setNibble := func(arr []byte, index int, v byte) {
		if index%2 == 0 {
			arr[index/2] = (arr[index/2] & 0xF0) | (v & 0x0F)
		} else {
			arr[index/2] = (arr[index/2] & 0x0F) | (v << 4)
		}
	}

	for lx := 0; lx < 16; lx++ {
		for lz := 0; lz < 16; lz++ {
			top := byte(0)
			for y := 0; y < CHUNK_HEIGHT; y++ {
				b := ch.GetBlock(lx, y, lz)
				idx := lx*CHUNK_HEIGHT*16 + lz*CHUNK_HEIGHT + y

				blocks[idx] = b.TypeId
				setNibble(data, idx, b.Metadata)
				setNibble(skyLight, idx, b.SkyLight)
				setNibble(blockLight, idx, b.Light)

				if b.SkyLight > 0 || b.TypeId != 0 {
					top = byte(y + 1)
				}
			}
			heightMap[lz*16+lx] = top
		}
	}

	level := mcregion.NewCompound()
	level.Int("xPos", cx)
	level.Int("zPos", cz)
	level.Long("LastUpdate", tick)
	level.Byte("TerrainPopulated", 1)
	level.ByteArray("Blocks", blocks)
	level.ByteArray("Data", data)
	level.ByteArray("SkyLight", skyLight)
	level.ByteArray("BlockLight", blockLight)
	level.ByteArray("HeightMap", heightMap)
	level.EmptyList("Entities")

	if len(w.Containers.Chests) > 0 {
		var tileEntities []*mcregion.Compound
		for pos, inv := range w.Containers.Chests {
			chunkX := WorldToChunkCoord(pos.X)
			chunkZ := WorldToChunkCoord(pos.Z)
			if chunkX != cx || chunkZ != cz {
				continue
			}
			// Negative coord correction
			lx := pos.X & 15
			lz := pos.Z & 15
			worldX := cx*16 + lx
			worldZ := cz*16 + lz
			tileEntities = append(tileEntities, buildChestNBT(worldX, int32(pos.Y), worldZ, inv))
		}

		for pos, inv := range w.Containers.Furnaces {
			chunkX := WorldToChunkCoord(pos.X)
			chunkZ := WorldToChunkCoord(pos.Z)
			if chunkX != cx || chunkZ != cz {
				continue
			}
			lx := pos.X & 15
			lz := pos.Z & 15
			worldX := cx*16 + lx
			worldZ := cz*16 + lz
			tileEntities = append(tileEntities, buildFurnaceNBT(worldX, int32(pos.Y), worldZ, inv))
		}

		for pos, inv := range w.Containers.Dispensers {
			chunkX := WorldToChunkCoord(pos.X)
			chunkZ := WorldToChunkCoord(pos.Z)
			if chunkX != cx || chunkZ != cz {
				continue
			}
			lx := pos.X & 15
			lz := pos.Z & 15
			worldX := cx*16 + lx
			worldZ := cz*16 + lz
			tileEntities = append(tileEntities, buildDispenserNBT(worldX, int32(pos.Y), worldZ, inv))
		}

		level.CompoundList("TileEntities", tileEntities)
	} else {
		level.EmptyList("TileEntities")
	}

	level.EmptyList("TileTicks")

	root := mcregion.NewCompound()
	root.AddCompound("Level", level)
	return root
}

func SaveMcRegion(w *World, worldDir string) error {
	w.Mu.RLock()
	chunksSnapshot := make(map[ChunkCoord]*Chunk, len(w.chunks))
	for coord, ch := range w.chunks {
		chunksSnapshot[coord] = ch
	}
	tick := w.Tick
	w.Mu.RUnlock()

	byRegion := make(map[[2]int32]map[[2]int32]*mcregion.Compound)
	for coord, ch := range chunksSnapshot {
		if ch == nil {
			continue
		}
		rx, rz := coord.X>>5, coord.Z>>5
		lx, lz := coord.X&31, coord.Z&31
		rkey := [2]int32{rx, rz}
		if byRegion[rkey] == nil {
			byRegion[rkey] = make(map[[2]int32]*mcregion.Compound)
		}
		byRegion[rkey][[2]int32{lx, lz}] = w.buildChunkNBT(ch, coord.X, coord.Z, tick)
	}

	regionDir := filepath.Join(worldDir, "region")
	for rkey, chunks := range byRegion {
		name := mcregion.RegionFileName(rkey[0]*32, rkey[1]*32)
		if err := mcregion.WriteRegion(filepath.Join(regionDir, name), chunks); err != nil {
			return err
		}
	}

	return saveLevelDat(worldDir, tick)
}

func saveLevelDat(worldDir string, tick int64) error {
	data := mcregion.NewCompound()
	data.Long("Time", tick)
	data.Long("LastPlayed", time.Now().UnixMilli())
	data.Long("RandomSeed", 0)
	data.Int("SpawnX", 0)
	data.Int("SpawnY", 64)
	data.Int("SpawnZ", 0)
	data.String("LevelName", "world")
	data.Int("version", 19132) // Version introduced in Beta 1.3

	root := mcregion.NewCompound()
	root.AddCompound("Data", data)

	if err := os.MkdirAll(worldDir, 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(root.Root()); err != nil {
		return err
	}
	if err := gw.Close(); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(worldDir, "level.dat"), buf.Bytes(), 0o644)
}
