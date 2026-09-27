package level

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/mcregion"
	"github.com/leNicDev/retromc/player"
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
	comp.Short("BurnTime", int16(furnace.FuelRemain))
	comp.Short("CookTime", int16(furnace.Progress))
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
	comp.String("id", "Trap")
	comp.Int("x", x)
	comp.Int("y", y)
	comp.Int("z", z)
	comp.CompoundList("Items", items)
	return comp
}

// Uses chunk.GetBlock to build NBT chunk block by block
func (w *World) buildChunkNBT(ch *Chunk, cx, cz, dim int32, ents []*mcregion.Compound, tick int64) *mcregion.Compound {
	// Chunk.Data already uses the MCRegion index order and nibble packing.
	blocks := append([]byte(nil), ch.Data[:chunkBlocksAmount]...)
	data := append([]byte(nil), ch.Data[chunkMetaOffset:chunkMetaOffset+chunkNibbleCount]...)
	blockLight := append([]byte(nil), ch.Data[chunkLightOffset:chunkLightOffset+chunkNibbleCount]...)
	skyLight := append([]byte(nil), ch.Data[chunkSkyOffset:chunkSkyOffset+chunkNibbleCount]...)

	heightMap := make([]byte, 256)
	for lx := 0; lx < 16; lx++ {
		for lz := 0; lz < 16; lz++ {
			base := lx*CHUNK_HEIGHT*16 + lz*CHUNK_HEIGHT
			top := 0
			for y := CHUNK_HEIGHT - 1; y >= 0; y-- {
				if constants.LightOpacity[blocks[base+y]] != 0 {
					top = y + 1
					break
				}
			}
			heightMap[lz*16+lx] = byte(top)
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
	if len(ents) > 0 {
		level.CompoundList("Entities", ents)
	} else {
		level.EmptyList("Entities")
	}

	var tileEntities []*mcregion.Compound
	inChunk := func(k BlockKey) bool {
		return k.Dim == dim && WorldToChunkCoord(k.X) == cx && WorldToChunkCoord(k.Z) == cz
	}
	for pos, inv := range w.Containers.Chests {
		if inChunk(pos) {
			tileEntities = append(tileEntities, buildChestNBT(pos.X, int32(pos.Y), pos.Z, inv))
		}
	}
	for pos, inv := range w.Containers.Furnaces {
		if inChunk(pos) {
			tileEntities = append(tileEntities, buildFurnaceNBT(pos.X, int32(pos.Y), pos.Z, inv))
		}
	}
	for pos, inv := range w.Containers.Dispensers {
		if inChunk(pos) {
			tileEntities = append(tileEntities, buildDispenserNBT(pos.X, int32(pos.Y), pos.Z, inv))
		}
	}
	if len(tileEntities) > 0 {
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
	snap := snapshotChunks(w)

	go func() {
		if err := saveChunkSnapshot(w, worldDir, snap); err != nil {
			log.Println("Failed to save world:", err)
		}
	}()

	return nil
}

func SaveMcRegionSync(w *World, worldDir string) error {
	return saveChunkSnapshot(w, worldDir, snapshotChunks(w))
}

type chunkSnapshot struct {
	oChunks, nChunks map[ChunkCoord]*Chunk
	oEnts, nEnts     map[ChunkCoord][]*mcregion.Compound
	tick             int64
}

// snapshotChunks must run on the game loop; entity NBT is built here, not in the save goroutine
func snapshotChunks(w *World) chunkSnapshot {
	oChunks := make(map[ChunkCoord]*Chunk, len(w.oChunks))
	for coord, ch := range w.oChunks {
		oChunks[coord] = ch
	}
	nChunks := make(map[ChunkCoord]*Chunk, len(w.nChunks))
	for coord, ch := range w.nChunks {
		nChunks[coord] = ch
	}
	snap := chunkSnapshot{
		oChunks: oChunks,
		nChunks: nChunks,
		oEnts:   w.CaptureEntities(oChunks, 0, nil),
		nEnts:   w.CaptureEntities(nChunks, -1, nil),
		tick:    w.Tick,
	}
	w.RelightForSave(oChunks, snap.oEnts, 0)
	w.RelightForSave(nChunks, snap.nEnts, -1)
	return snap
}

// SetBlock copies the placed block's own light values into the chunk, so stored light
// is stale until the chunk is relit.
func (w *World) RelightForSave(chunks map[ChunkCoord]*Chunk, ents map[ChunkCoord][]*mcregion.Compound, dim int32) {
	for coord, ch := range chunks {
		if _, hasEnts := ents[coord]; ch != nil && (ch.HasChanged || hasEnts) {
			w.RelightChunk(coord.X, coord.Z, dim, ch)
		}
	}
}

func saveChunkSnapshot(w *World, worldDir string, snap chunkSnapshot) error {
	if err := saveChunksToRegion(w, worldDir, 0, snap.oChunks, snap.oEnts, snap.tick); err != nil {
		return fmt.Errorf("saving overworld region: %w", err)
	}
	if err := saveChunksToRegion(w, filepath.Join(worldDir, "DIM-1"), -1, snap.nChunks, snap.nEnts, snap.tick); err != nil {
		return fmt.Errorf("saving nether region: %w", err)
	}
	return saveLevelDat(worldDir, w.Seed, snap.tick)
}

// Region and level.dat saves are read-modify-write and can run from several goroutines.
var saveMu sync.Mutex

func saveChunksToRegion(w *World, dir string, dim int32, chunks map[ChunkCoord]*Chunk, ents map[ChunkCoord][]*mcregion.Compound, tick int64) error {
	if len(chunks) == 0 {
		return nil
	}

	byRegion := make(map[[2]int32]map[[2]int32]*mcregion.Compound)
	for coord, ch := range chunks {
		if ch == nil {
			continue
		}

		chunkEnts, hasEnts := ents[coord]
		if !ch.HasChanged && !hasEnts {
			continue
		}
		rx, rz := coord.X>>5, coord.Z>>5
		lx, lz := coord.X&31, coord.Z&31
		rkey := [2]int32{rx, rz}
		if byRegion[rkey] == nil {
			byRegion[rkey] = make(map[[2]int32]*mcregion.Compound)
		}
		byRegion[rkey][[2]int32{lx, lz}] = w.buildChunkNBT(ch, coord.X, coord.Z, dim, chunkEnts, tick)
	}

	saveMu.Lock()
	defer saveMu.Unlock()

	regionDir := filepath.Join(dir, "region")
	if err := os.MkdirAll(regionDir, 0o755); err != nil {
		return err
	}
	for rkey, chunks := range byRegion {
		name := mcregion.RegionFileName(rkey[0]*32, rkey[1]*32)
		path := filepath.Join(regionDir, name)

		rawChunks, err := mcregion.ReadRegionRaw(path)
		if err != nil {
			return fmt.Errorf("reading existing region %s: %w", path, err)
		}
		if err := mcregion.WriteRegion(path, chunks, rawChunks); err != nil {
			return err
		}
	}
	return nil
}

func SaveChunks(w *World, worldDir string, chunks map[ChunkCoord]*Chunk, ents map[ChunkCoord][]*mcregion.Compound, dimension int32, tick int64) error {
	if len(chunks) == 0 {
		return nil
	}

	dir := worldDir
	if dimension == -1 {
		dir = filepath.Join(worldDir, "DIM-1")
	}

	if err := saveChunksToRegion(w, dir, dimension, chunks, ents, tick); err != nil {
		return err
	}
	return saveLevelDat(worldDir, w.Seed, tick)
}

func saveLevelDat(worldDir string, seed, tick int64) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	var sizeOnDisk int64
	filepath.WalkDir(worldDir, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if info, err := d.Info(); err == nil {
				sizeOnDisk += info.Size()
			}
		}
		return nil
	})

	data := mcregion.NewCompound()
	data.Long("RandomSeed", seed)
	data.Int("SpawnX", int32(player.SpawnX))
	data.Int("SpawnY", int32(player.SpawnY))
	data.Int("SpawnZ", int32(player.SpawnZ))
	data.Int("rainTime", 0)
	data.Int("thunderTime", 0)
	data.Byte("raining", 0)
	data.Byte("thundering", 0)
	data.Long("Time", tick)
	data.Long("LastPlayed", time.Now().UnixMilli())
	data.Long("SizeOnDisk", sizeOnDisk)
	data.String("LevelName", "world")
	data.Int("version", 19132) // McRegion format version

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

	path := filepath.Join(worldDir, "level.dat")
	if old, err := os.ReadFile(path); err == nil {
		if err := os.WriteFile(path+"_old", old, 0o644); err != nil {
			return err
		}
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func (w *World) readChunkFromNBT(lvl *mcregion.Tag, cx, cz, dim int32) (*Chunk, error) {
	blocks := lvl.Get("Blocks").ByteArr
	data := lvl.Get("Data").ByteArr
	skyLight := lvl.Get("SkyLight").ByteArr
	blockLight := lvl.Get("BlockLight").ByteArr

	if len(blocks) != 16*CHUNK_HEIGHT*16 {
		return nil, fmt.Errorf("chunk (%d,%d): unexpected Blocks length %d", cx, cz, len(blocks))
	}

	getNibble := func(arr []byte, index int) byte {
		if index%2 == 0 {
			return arr[index/2] & 0x0F
		}
		return (arr[index/2] >> 4) & 0x0F
	}

	c := NewChunk()
	c.X = cx * CHUNK_SIZE_X
	c.Z = cz * CHUNK_SIZE_Z

	for lx := 0; lx < 16; lx++ {
		for lz := 0; lz < 16; lz++ {
			for y := 0; y < CHUNK_HEIGHT; y++ {
				idx := lx*CHUNK_HEIGHT*16 + lz*CHUNK_HEIGHT + y
				b := constants.WBlock{
					TypeId:   blocks[idx],
					Metadata: getNibble(data, idx),
					SkyLight: getNibble(skyLight, idx),
					Light:    getNibble(blockLight, idx),
				}

				c.SetBlock(lx, y, lz, b)
			}
		}
	}

	if el := lvl.Get("Entities"); el != nil && len(el.List) > 0 {
		c.PendingEntities = el.List
	}

	teCount := 0
	if teList := lvl.Get("TileEntities"); teList != nil {
		for _, te := range teList.List {
			id := te.Get("id")
			if id == nil {
				continue
			}
			x := te.Get("x").IntVal
			y := te.Get("y").IntVal
			z := te.Get("z").IntVal
			key := BlockKey{X: x, Y: byte(y), Z: z, Dim: dim}

			switch id.StrVal {
			case "Chest":
				chest := inventory.NewChest(CHEST_SIZE)
				chest.SetPosition(x, y, z)
				loadItemSlots(te, chest.Items)
				w.Containers.Chests[key] = &chest
			case "Furnace":
				furnace := inventory.NewFurnace()
				furnace.SetPosition(x, y, z)
				furnace.Dim = dim
				loadItemSlots(te, furnace.Items[:])
				if t := te.Get("BurnTime"); t != nil && t.ShortVal > 0 {
					furnace.FuelRemain = int(t.ShortVal)
					furnace.IsBurning = true
					// Vanilla also derives the fuel bar's max from the item left in the fuel slot.
					furnace.MaxFuel = inventory.FuelBurnTime(furnace.Items[1].TypeId)
					if furnace.MaxFuel < furnace.FuelRemain {
						furnace.MaxFuel = furnace.FuelRemain
					}
				}
				if t := te.Get("CookTime"); t != nil {
					furnace.Progress = int(t.ShortVal)
				}
				w.Containers.Furnaces[key] = furnace
			case "Trap":
				dispenser := inventory.NewDispenser()
				loadItemSlots(te, dispenser.Items[:])
				w.Containers.Dispensers[key] = dispenser
			}
			teCount++
		}
	}
	return &c, nil
}

func loadItemSlots(te *mcregion.Tag, slots []inventory.Item) {
	items := te.Get("Items")
	if items == nil {
		return
	}
	for _, item := range items.List {
		slot := int(item.Get("Slot").ByteVal)
		if slot < 0 || slot >= len(slots) {
			continue
		}
		slots[slot] = inventory.Item{
			TypeId:   item.Get("id").ShortVal,
			Count:    item.Get("Count").ByteVal,
			Metadata: uint16(item.Get("Damage").ShortVal),
		}
	}
}

type PlayerInventorySlot struct {
	Slot   byte
	ItemID int16
	Damage int16
	Count  byte
}

type PlayerData struct {
	X, Y, Z                   float64
	MotionX, MotionY, MotionZ float64
	Yaw, Pitch                float32
	FallDistance              float32
	Health                    int16
	Air                       int16
	Fire                      int16
	OnGround                  byte
	Sleeping                  byte
	SleepTimer                int16
	Dimension                 int32
	DeathTime                 int16
	HurtTime                  int16
	AttackTime                int16
	Inventory                 []PlayerInventorySlot
}

func playerFilePath(worldDir, name string) string {
	return filepath.Join(worldDir, "players", name+".dat")
}

func SavePlayerData(worldDir, name string, data *PlayerData) error {
	root := buildPlayerNBT(data)

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(root.Root()); err != nil {
		return err
	}
	if err := gw.Close(); err != nil {
		return err
	}

	dir := filepath.Join(worldDir, "players")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	finalPath := playerFilePath(worldDir, name)
	tmpPath := finalPath + ".tmp"
	if err := os.WriteFile(tmpPath, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, finalPath)
}

func buildPlayerNBT(data *PlayerData) *mcregion.Compound {
	root := mcregion.NewCompound()

	root.DoubleList("Pos", []float64{data.X, data.Y, data.Z})
	root.DoubleList("Motion", []float64{data.MotionX, data.MotionY, data.MotionZ})
	root.FloatList("Rotation", []float32{data.Yaw, data.Pitch})
	root.Float("FallDistance", data.FallDistance)
	root.Short("Health", data.Health)
	root.Short("Air", data.Air)
	root.Short("Fire", data.Fire)
	root.Byte("OnGround", data.OnGround)
	root.Byte("Sleeping", data.Sleeping)
	root.Short("SleepTimer", data.SleepTimer)
	root.Int("Dimension", data.Dimension)
	root.Short("DeathTime", data.DeathTime)
	root.Short("HurtTime", data.HurtTime)
	root.Short("AttackTime", data.AttackTime)

	var items []*mcregion.Compound
	for _, slot := range data.Inventory {
		item := mcregion.NewCompound()
		item.Short("id", slot.ItemID)
		item.Short("Damage", slot.Damage)
		item.Byte("Count", slot.Count)
		item.Byte("Slot", slot.Slot)
		items = append(items, item)
	}
	root.CompoundList("Inventory", items)

	return root
}

// LoadPlayerData reads worldDir/players/<name>.dat. If the file doesn't
// exist, it returns a fresh PlayerData (NewPlayerData()) rather than an
// error — matching the C++ reference's "create on first join" behavior.
func LoadPlayerData(worldDir, name string) (*PlayerData, error) {
	path := playerFilePath(worldDir, name)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fresh := NewPlayerData()
			if saveErr := SavePlayerData(worldDir, name, fresh); saveErr != nil {
				return nil, saveErr
			}
			return fresh, nil
		}
		return nil, err
	}

	gr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	decompressed, err := io.ReadAll(gr)
	if err != nil {
		return nil, err
	}

	root, err := mcregion.ParseRoot(decompressed)
	if err != nil {
		return nil, err
	}

	return playerDataFromNBT(root)
}

func NewPlayerData() *PlayerData {
	return &PlayerData{
		X: -1, Y: -1000000, Z: -1,
		Health: 20,
		Air:    300,
		Fire:   -20,
	}
}

func playerDataFromNBT(root *mcregion.Tag) (*PlayerData, error) {
	data := NewPlayerData()

	if pos := root.Get("Pos"); pos != nil && len(pos.List) == 3 {
		data.X = pos.List[0].DoubleVal
		data.Y = pos.List[1].DoubleVal
		data.Z = pos.List[2].DoubleVal
	}
	if motion := root.Get("Motion"); motion != nil && len(motion.List) == 3 {
		data.MotionX = motion.List[0].DoubleVal
		data.MotionY = motion.List[1].DoubleVal
		data.MotionZ = motion.List[2].DoubleVal
	}
	if rot := root.Get("Rotation"); rot != nil && len(rot.List) == 2 {
		data.Yaw = rot.List[0].FloatVal
		data.Pitch = rot.List[1].FloatVal
	}
	if t := root.Get("FallDistance"); t != nil {
		data.FallDistance = t.FloatVal
	}
	if t := root.Get("Health"); t != nil {
		data.Health = t.ShortVal
	}
	if t := root.Get("Air"); t != nil {
		data.Air = t.ShortVal
	}
	if t := root.Get("Fire"); t != nil {
		data.Fire = t.ShortVal
	}
	if t := root.Get("OnGround"); t != nil {
		data.OnGround = t.ByteVal
	}
	if t := root.Get("Sleeping"); t != nil {
		data.Sleeping = t.ByteVal
	}
	if t := root.Get("SleepTimer"); t != nil {
		data.SleepTimer = t.ShortVal
	}
	if t := root.Get("Dimension"); t != nil {
		data.Dimension = t.IntVal
	}
	if t := root.Get("DeathTime"); t != nil {
		data.DeathTime = t.ShortVal
	}
	if t := root.Get("HurtTime"); t != nil {
		data.HurtTime = t.ShortVal
	}
	if t := root.Get("AttackTime"); t != nil {
		data.AttackTime = t.ShortVal
	}
	if inv := root.Get("Inventory"); inv != nil {
		for _, item := range inv.List {
			slot := PlayerInventorySlot{}
			if s := item.Get("Slot"); s != nil {
				slot.Slot = s.ByteVal
			}
			if id := item.Get("id"); id != nil {
				slot.ItemID = id.ShortVal
			}
			if dmg := item.Get("Damage"); dmg != nil {
				slot.Damage = dmg.ShortVal
			}
			if cnt := item.Get("Count"); cnt != nil {
				slot.Count = cnt.ByteVal
			}
			data.Inventory = append(data.Inventory, slot)
		}
	}

	return data, nil
}

func ToPlayerData(p *player.Player) *PlayerData {
	var items []PlayerInventorySlot
	for i, item := range p.Inventory.Items {
		nbtSlot, ok := windowToNbtSlot(i)
		if !ok || item.TypeId <= 0 || item.Count == 0 {
			continue
		}
		items = append(items, PlayerInventorySlot{
			Slot:   nbtSlot,
			ItemID: item.TypeId,
			Damage: int16(item.Metadata),
			Count:  item.Count,
		})
	}

	var onGround byte
	if p.OnGround {
		onGround = 1
	}
	data := NewPlayerData()
	data.X, data.Y, data.Z = p.X, p.Y, p.Z
	data.MotionX, data.MotionY, data.MotionZ = p.Vx, p.Vy, p.Vz
	data.Yaw, data.Pitch = p.Yaw, p.Pitch
	data.FallDistance = float32(p.FallDistance)
	data.OnGround = onGround
	data.Health = p.HP
	data.Inventory = items
	data.Dimension = p.Dimension
	return data
}

// Beta NBT player slots: 0-8 hotbar, 9-35 main, 100-103 armor (boots..helmet).
// The player window uses 5-8 armor (helmet..boots), 9-35 main, 36-44 hotbar.
func windowToNbtSlot(w int) (byte, bool) {
	switch {
	case w >= 9 && w <= 35:
		return byte(w), true
	case w >= 36 && w <= 44:
		return byte(w - 36), true
	case w >= 5 && w <= 8:
		return byte(108 - w), true
	}
	return 0, false
}

func nbtToWindowSlot(s byte) (int, bool) {
	switch {
	case s <= 8:
		return int(s) + 36, true
	case s <= 35:
		return int(s), true
	case s >= 100 && s <= 103:
		return 108 - int(s), true
	}
	return 0, false
}

func ApplyPlayerData(p *player.Player, data *PlayerData) {
	size := len(p.Inventory.Items)
	items := make([]inventory.Item, size)
	for i := range items {
		items[i] = inventory.NewItem(-1, 0, 0)
	}
	for _, saved := range data.Inventory {
		slot, ok := nbtToWindowSlot(saved.Slot)
		if !ok || slot >= size || saved.ItemID <= 0 || saved.Count == 0 {
			continue
		}
		items[slot] = inventory.Item{
			TypeId:   saved.ItemID,
			Metadata: uint16(saved.Damage),
			Count:    saved.Count,
		}
	}
	//log.Printf("Stored Coords x=%f, y=%f, z=%f", data.X, data.Y, data.Z)
	p.X, p.Y, p.Z = data.X, data.Y, data.Z
	p.Vx, p.Vy, p.Vz = data.MotionX, data.MotionY, data.MotionZ
	p.Yaw, p.Pitch = data.Yaw, data.Pitch
	p.FallDistance = float64(data.FallDistance)
	p.OnGround = data.OnGround != 0
	p.HP = data.Health
	p.Inventory.Items = items
	p.Dimension = data.Dimension
}
