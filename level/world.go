package level

import (
	"fmt"
	"sync"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/player"
)

// ChunkCoord is the map key for a chunk's position in the world.
type ChunkCoord struct {
	X, Z int32
}

func (c *ChunkCoord) String() string {
	return fmt.Sprintf("%d-%d", c.X, c.Z)
}

// BlockKey is the map key for a single block's world position.
type BlockKey struct {
	X int32
	Y byte
	Z int32
}

type Entity interface {
	GetName() string
	GetPosition() (float64, float64, float64)
	SetPosition(x, y, z float64)
	IsRideable() bool
	GetEntityId() int32
	IsPlayer() bool
	SetHP(hp int16)
	GetHP() int16
}

const VIEW_DISTANCE = 4

func (w *World) UnloadUnusedChunks() {
	w.Mu.Lock()
	defer w.Mu.Unlock()

	wanted := make(map[ChunkCoord]struct{})
	for _, pl := range w.Players {
		cx := WorldToChunkCoord(int32(pl.X))
		cz := WorldToChunkCoord(int32(pl.Z))
		for dx := -VIEW_DISTANCE; dx <= VIEW_DISTANCE; dx++ {
			for dz := -VIEW_DISTANCE; dz <= VIEW_DISTANCE; dz++ {
				wanted[ChunkCoord{X: cx + int32(dx), Z: cz + int32(dz)}] = struct{}{}
			}
		}
	}

	for coord := range w.chunks {
		if _, ok := wanted[coord]; !ok {
			delete(w.chunks, coord)
		}
	}
}

func (w *World) SnapshotEntities() []Entity {
	w.Mu.RLock()
	defer w.Mu.RUnlock()
	snapshot := make([]Entity, 0, len(w.Entities))
	for _, e := range w.Entities {
		snapshot = append(snapshot, e)
	}
	return snapshot
}

func (w *World) Size() int64 {
	w.Mu.Lock()
	defer w.Mu.Unlock()

	var total int64
	for _, c := range w.chunks {
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

type WorldType int

const (
	Template WorldType = iota
	Empty
	SkyGrid
)

type DroppedItem struct {
	EntityId    int32
	ItemId      int32
	Amount      byte
	Metadata    byte
	X, Y, Z     int32
	PickupDelay int32
}

// World holds all loaded chunks and is the single source of truth for block state.
type World struct {
	Mu           sync.RWMutex
	chunks       map[ChunkCoord]*Chunk
	changes      map[BlockKey]Block
	Tick         int64
	Players      map[int32]*player.Player
	Entities     map[int32]Entity
	Fallables    map[BlockKey]struct{} // Set of sand and gravel blocks that need to be checked for falling each tick.
	EntityCount  int32
	WaterSources map[BlockKey]byte
	FlowingWater map[BlockKey]byte
	LavaSources  map[BlockKey]byte
	FlowingLava  map[BlockKey]byte
	DroppedItems map[int32]*DroppedItem
	WorldType    WorldType

	Growables map[BlockKey]Growable
	TickSpeed int64
}

func NewWorld(worldType WorldType) *World {
	return &World{
		WorldType:    worldType,
		chunks:       make(map[ChunkCoord]*Chunk),
		changes:      make(map[BlockKey]Block),
		EntityCount:  0,
		Players:      make(map[int32]*player.Player),
		Entities:     make(map[int32]Entity),
		Fallables:    make(map[BlockKey]struct{}),
		WaterSources: make(map[BlockKey]byte),
		FlowingWater: make(map[BlockKey]byte),
		LavaSources:  make(map[BlockKey]byte),
		FlowingLava:  make(map[BlockKey]byte),
		Growables:    make(map[BlockKey]Growable),
		TickSpeed:    1,
		DroppedItems: make(map[int32]*DroppedItem),
	}
}

func (w *World) GetFirstPlayerByName(name string) *player.Player {
	for _, p := range w.Players {
		if p.Username == name {
			return p
		}
	}
	return nil
}

func (w *World) AddDroppedItem(x, y, z int32, itemId int32, amount, meta byte, pickupDelay int32) int32 {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	entityId := w.NextEntityId()
	w.DroppedItems[entityId] = &DroppedItem{EntityId: entityId, ItemId: itemId, Amount: amount, Metadata: meta, X: x, Y: y, Z: z, PickupDelay: pickupDelay}
	return entityId
}

func (w *World) RemoveDroppedItem(entityId int32) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	delete(w.DroppedItems, entityId)
}

func (w *World) AddFallable(x int32, y byte, z int32) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	w.Fallables[BlockKey{x, y, z}] = struct{}{}
}

func (w *World) RemoveFallable(x int32, y byte, z int32) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	delete(w.Fallables, BlockKey{x, y, z})
}

func (w *World) CleanUpFallable() {
	for key := range w.Fallables {
		block := w.GetBlock(key.X, key.Y, key.Z)
		if block.TypeId != byte(constants.Sand.Value) && block.TypeId != byte(constants.Gravel.Value) {
			w.RemoveFallable(key.X, key.Y, key.Z)
		}
	}
}

func (w *World) LoadChunks() map[ChunkCoord]*Chunk {
	return w.chunks
}

// ChunkExists reports whether the chunk at (cx, cz) has already been loaded/generated.
func (w *World) ChunkExists(cx, cz int32) bool {
	w.Mu.RLock()
	defer w.Mu.RUnlock()
	_, ok := w.chunks[ChunkCoord{cx, cz}]
	return ok
}

// GetOrCreateChunk returns the chunk at (cx, cz), generating it if it doesn't exist yet.
func (w *World) GetOrCreateChunk(cx, cz int32, worldType WorldType) *Chunk {
	w.Mu.Lock()
	defer w.Mu.Unlock()

	key := ChunkCoord{cx, cz}
	if c, ok := w.chunks[key]; ok {
		return c
	}

	c := NewChunk(worldType)
	c.X = cx * CHUNK_SIZE_X
	c.Z = cz * CHUNK_SIZE_Z

	// Replay any persisted block changes that fall in this chunk.
	for k, b := range w.changes {
		if WorldToChunkCoord(k.X) == cx && WorldToChunkCoord(k.Z) == cz {
			lx := WorldToLocalCoord(k.X)
			lz := WorldToLocalCoord(k.Z)
			c.SetBlock(lx, int(k.Y), lz, b)
			w.SetGrowable(b, k)
			if b.TypeId == byte(constants.Sand.Value) || b.TypeId == byte(constants.Gravel.Value) {
				w.Fallables[k] = struct{}{}
			}

			if b.TypeId == byte(constants.Wheat.Value) {
				w.Growables[k] = &Wheat{StartTick: w.Tick, State: b.Metadata}
			}

			if b.IsStillWater() {
				w.WaterSources[k] = b.Metadata
			}
			if b.IsStillLava() {
				w.LavaSources[k] = b.Metadata
			}
			if b.IsFlowingWater() {
				w.FlowingWater[k] = b.Metadata
			}
			if b.IsFlowingLava() {
				w.FlowingLava[k] = b.Metadata
			}
		}
	}

	w.chunks[key] = &c
	return &c
}

// SetBlock updates a single block in the world using world-space coordinates.
func (w *World) SetBlock(worldX int32, worldY byte, worldZ int32, block Block) {
	cx := WorldToChunkCoord(worldX)
	cz := WorldToChunkCoord(worldZ)
	chunk := w.GetOrCreateChunk(cx, cz, Empty)

	lx := WorldToLocalCoord(worldX)
	lz := WorldToLocalCoord(worldZ)
	chunk.SetBlock(lx, int(worldY), lz, block)

	// in-memory persistence
	w.Mu.Lock()
	key := BlockKey{worldX, worldY, worldZ}
	w.SetGrowable(block, key)

	w.changes[key] = block
	if block.IsStillWater() {
		w.WaterSources[key] = block.Metadata
		delete(w.FlowingWater, key)
	} else if block.IsFlowingWater() {
		w.FlowingWater[key] = block.Metadata
		delete(w.WaterSources, key)
	} else if block.IsStillLava() {
		w.LavaSources[key] = block.Metadata
		delete(w.FlowingLava, key)
	} else if block.IsFlowingLava() {
		w.FlowingLava[key] = block.Metadata
		delete(w.LavaSources, key)
	} else {
		delete(w.WaterSources, key)
		delete(w.FlowingWater, key)
		delete(w.LavaSources, key)
		delete(w.FlowingLava, key)
	}
	w.Mu.Unlock()
}

func (w *World) GetBlock(worldX int32, worldY byte, worldZ int32) Block {
	cx := WorldToChunkCoord(worldX)
	cz := WorldToChunkCoord(worldZ)
	chunk := w.GetOrCreateChunk(cx, cz, Empty)

	lx := WorldToLocalCoord(worldX)
	lz := WorldToLocalCoord(worldZ)
	return chunk.GetBlock(lx, int(worldY), lz)
}

func (w *World) NextEntityId() int32 {
	w.EntityCount++
	return w.EntityCount
}

func (w *World) AddPlayer(p *player.Player) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	p.EntityId = int(w.NextEntityId())
	w.Players[int32(p.EntityId)] = p
	w.Entities[int32(p.EntityId)] = p
}

func (w *World) AddRidable(entityId, ownerEntityId int32, x, y, z, vx, vy, vz float64, objectType byte) {
	r := entities.RideableEntity{
		EntityId:      entityId,
		OwnerEntityId: ownerEntityId,
		X:             x,
		Y:             y,
		Z:             z,
		VelocityX:     vx,
		VelocityY:     vy,
		VelocityZ:     vz,
		ObjectType:    objectType,
		HP:            4,
	}
	w.Mu.Lock()
	defer w.Mu.Unlock()
	w.Entities[int32(entityId)] = &r
}

func (w *World) AddEntity(e Entity) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	w.Entities[e.GetEntityId()] = e
}

func (w *World) RemovePlayer(p *player.Player) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	delete(w.Players, int32(p.EntityId))
}

func (w *World) RemoveEntity(entityId int32) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	delete(w.Entities, entityId)
}

type SetTimePacket struct {
	Time int64
}

func (p *SetTimePacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.TimeUpdate)
	w.WriteInt64(p.Time)
	return w.Bytes()
}

func (w *World) BroadcastTime() {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	packet := SetTimePacket{Time: w.Tick}
	data := packet.Serialize()
	for _, pl := range w.Players {
		if pl.LoggedIn {
			pl.Connection.Write(data)
		}
	}
}

// BroadcastPacket sends raw pre-serialized packet data to all logged-in players.
func (w *World) BroadcastPacket(data []byte) {
	w.Mu.RLock()
	defer w.Mu.RUnlock()
	for _, pl := range w.Players {
		if pl.LoggedIn {
			pl.Connection.Write(data)
		}
	}
}

func (w *World) MulticastPacket(data []byte, exclude *player.Player) {
	w.Mu.RLock()
	defer w.Mu.RUnlock()
	for _, pl := range w.Players {
		if pl.LoggedIn && pl != exclude {
			pl.Connection.Write(data)
		}
	}
}

func (w *World) ForEachPlayer(fn func(*player.Player)) {
	w.Mu.RLock()
	defer w.Mu.RUnlock()
	for _, pl := range w.Players {
		if pl.LoggedIn {
			fn(pl)
		}
	}
}
