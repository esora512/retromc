package level

import (
	"sync"
	"fmt"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/player"
)

// ChunkCoord is the map key for a chunk's position in the world.
type ChunkCoord struct {
	X, Z int32
}

func (c*ChunkCoord) String() string {
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

func (w *World) SnapshotEntities() []Entity {
	w.mu.RLock()
	defer w.mu.RUnlock()
	snapshot := make([]Entity, 0, len(w.Entities))
	for _, e := range w.Entities {
		snapshot = append(snapshot, e)
	}
	return snapshot
}

type WorldType int

const (
	Template WorldType = iota
	Empty
	SkyGrid
)

// World holds all loaded chunks and is the single source of truth for block state.
type World struct {
	mu           sync.RWMutex
	chunks       map[ChunkCoord]*Chunk
	changes      map[BlockKey]Block
	Tick         int64
	Players      map[int32]*player.Player
	Entities     map[int32]Entity
	EntityCount  int32
	WaterSources map[BlockKey]byte
	FlowingWater map[BlockKey]byte
	LavaSources  map[BlockKey]byte
	FlowingLava  map[BlockKey]byte
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
		WaterSources: make(map[BlockKey]byte),
		FlowingWater: make(map[BlockKey]byte),
		LavaSources:  make(map[BlockKey]byte),
		FlowingLava:  make(map[BlockKey]byte),
		Growables:    make(map[BlockKey]Growable),
		TickSpeed:    1,
	}
}

func (w *World) LoadChunks() map[ChunkCoord]*Chunk {
	return w.chunks
}

// ChunkExists reports whether the chunk at (cx, cz) has already been loaded/generated.
func (w *World) ChunkExists(cx, cz int32) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	_, ok := w.chunks[ChunkCoord{cx, cz}]
	return ok
}

// GetOrCreateChunk returns the chunk at (cx, cz), generating it if it doesn't exist yet.
func (w *World) GetOrCreateChunk(cx, cz int32, worldType WorldType) *Chunk {
	w.mu.Lock()
	defer w.mu.Unlock()

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
	w.mu.Lock()
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
	w.mu.Unlock()
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
	w.mu.Lock()
	defer w.mu.Unlock()
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
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Entities[int32(entityId)] = &r
}

func (w *World) RemovePlayer(p *player.Player) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.Players, int32(p.EntityId))
}

func (w *World) RemoveEntity(entityId int32) {
	w.mu.Lock()
	defer w.mu.Unlock()
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
	w.mu.Lock()
	defer w.mu.Unlock()
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
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, pl := range w.Players {
		if pl.LoggedIn {
			pl.Connection.Write(data)
		}
	}
}

func (w *World) MulticastPacket(data []byte, exclude *player.Player) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, pl := range w.Players {
		if pl.LoggedIn && pl != exclude {
			pl.Connection.Write(data)
		}
	}
}

func (w *World) ForEachPlayer(fn func(*player.Player)) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, pl := range w.Players {
		if pl.LoggedIn {
			fn(pl)
		}
	}
}
