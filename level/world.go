package level

import (
	"fmt"
	"sync"

	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/player"
)

// ChunkCoord is the map key for a chunk's position in the world.
type ChunkCoord struct {
	X, Z int32
}

type IEntity interface {
	GetName() string
	GetPosition() (float64, float64, float64)
}

type RideableEntity struct {
	Entityid  int32
	X         float64
	Y         float64
	Z         float64
	VelocityX float64
	VelocityY float64
	VelocityZ float64
}

func (r *RideableEntity) GetPosition() (float64, float64, float64) {
	return r.X, r.Y, r.Z
}

func (r *RideableEntity) GetName() string {
	return fmt.Sprintf("Entity %d", r.Entityid)
}

// World holds all loaded chunks and is the single source of truth for block state.
type World struct {
	mu          sync.RWMutex
	chunks      map[ChunkCoord]*Chunk
	Tick        int64
	Players     map[int32]*player.Player
	Entities    map[int32]IEntity
	EntityCount int32
}

func NewWorld() *World {
	return &World{chunks: make(map[ChunkCoord]*Chunk), EntityCount: 0, Players: make(map[int32]*player.Player), Entities: make(map[int32]IEntity)}
}

// ChunkExists reports whether the chunk at (cx, cz) has already been loaded/generated.
func (w *World) ChunkExists(cx, cz int32) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	_, ok := w.chunks[ChunkCoord{cx, cz}]
	return ok
}

// GetOrCreateChunk returns the chunk at (cx, cz), generating it if it doesn't exist yet.
func (w *World) GetOrCreateChunk(cx, cz int32) *Chunk {
	w.mu.Lock()
	defer w.mu.Unlock()

	key := ChunkCoord{cx, cz}
	if c, ok := w.chunks[key]; ok {
		return c
	}

	c := NewChunk()
	c.X = cx * CHUNK_SIZE_X
	c.Z = cz * CHUNK_SIZE_Z
	w.chunks[key] = &c
	return &c
}

// SetBlock updates a single block in the world using world-space coordinates.
func (w *World) SetBlock(worldX int32, worldY byte, worldZ int32, block Block) {
	cx := WorldToChunkCoord(worldX)
	cz := WorldToChunkCoord(worldZ)
	chunk := w.GetOrCreateChunk(cx, cz)

	lx := WorldToLocalCoord(worldX)
	lz := WorldToLocalCoord(worldZ)
	chunk.SetBlock(lx, int(worldY), lz, block)
}

func (w *World) GetBlock(worldX int32, worldY byte, worldZ int32) Block {
	cx := WorldToChunkCoord(worldX)
	cz := WorldToChunkCoord(worldZ)
	chunk := w.GetOrCreateChunk(cx, cz)

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

func (w *World) AddRidable(entityId int32, x, y, z, vx, vy, vz float64) {
	r := RideableEntity{
		Entityid:  entityId,
		X:         x,
		Y:         y,
		Z:         z,
		VelocityX: vx,
		VelocityY: vy,
		VelocityZ: vz,
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
