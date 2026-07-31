package level

import (
	"fmt"
	"log"
	"math"
	"net"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/mcregion"
	"github.com/leNicDev/retromc/player"
)

const VIEW_DISTANCE = 4

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
	GetLoggedIn() bool
}

func (w *World) GetPlayerByUsername(name string) (*player.Player, bool) {
	for _, pl := range w.Players {
		if pl.Username == name {
			return pl, true
		}
	}
	return nil, false
}

type EntityTracker struct {
	SpawnPlayer      func(pl *player.Player) []byte
	SpawnObject      func(e Entity) []byte
	DespawnEntity    func(id int32) []byte
	SetEquipment     func(pl *player.Player, send func([]byte) (int, error))
	visible          map[int32]map[int32]bool
	Mu               sync.Mutex
}

func NewEntityTracker(
	spawnPlayer func(pl *player.Player) []byte,
	spawnObject func(e Entity) []byte,
	despawnEntity func(id int32) []byte,
	setEquipment func(pl *player.Player, send func([]byte) (int, error)),
) *EntityTracker {
	return &EntityTracker{
		SpawnPlayer:      spawnPlayer,
		SpawnObject:      spawnObject,
		DespawnEntity:    despawnEntity,
		SetEquipment:     setEquipment,
		visible:          make(map[int32]map[int32]bool),
	}
}

func (et *EntityTracker) Remove(id int32) {
	et.Mu.Lock()
	defer et.Mu.Unlock()
	delete(et.visible, id)
	for _, seen := range et.visible {
		delete(seen, id)
	}
}

func (et *EntityTracker) Add(playerId int32, otherId int32) {
	et.Mu.Lock()
	defer et.Mu.Unlock()
	if et.visible[playerId] == nil {
		et.visible[playerId] = make(map[int32]bool)
	}
	et.visible[playerId][otherId] = true
}

func (et *EntityTracker) Manage(w *World) {
	et.Mu.Lock()
	defer et.Mu.Unlock()
	const distance = VIEW_DISTANCE * 8

	for _, viewer := range w.Players {
		viewerID := viewer.GetEntityId()

		if et.visible[viewerID] == nil {
			et.visible[viewerID] = make(map[int32]bool)
		}

		if !viewer.LoggedIn {
			continue
		}

		x1, _, z1 := viewer.GetPosition()

		for _, target := range w.Entities {
			targetID := target.GetEntityId()

			if viewerID == targetID {
				continue
			}

			if !target.GetLoggedIn() {
				continue
			}

			x2, _, z2 := target.GetPosition()

			dx := math.Abs(x1 - x2)
			dz := math.Abs(z1 - z2)

			isVisible := et.visible[viewerID][targetID]
			inRange := dx <= distance && dz <= distance
			alive := target.GetHP() > 0

			if !isVisible && inRange && alive {
				if target.IsPlayer() {
					if target.GetName() == viewer.Username {
						continue
					}
					t, _ := target.(*player.Player)
					viewer.Connection.Write(et.SpawnPlayer(t))
					et.SetEquipment(t, viewer.Connection.Write)
				} else {
					viewer.Connection.Write(et.SpawnObject(target))
				}
				et.visible[viewerID][targetID] = true
			} else if isVisible && (!inRange || !alive) {
				viewer.Connection.Write(et.DespawnEntity(targetID))
				delete(et.visible[viewerID], targetID)
			}
		}
	}
}

func (w *World) GetLoadedChunk(x, z int32) *Chunk {
	cx := WorldToChunkCoord(x)
	cz := WorldToChunkCoord(z)
	return w.chunks[ChunkCoord{cx, cz}]
}

func (w *World) GetRenderedChunks() []*Chunk {
	wanted := w.wantedChunks()
	chunks := make([]*Chunk, 0, len(wanted))
	for wa := range wanted {
		if c, ok := w.chunks[wa]; ok {
			chunks = append(chunks, c)
		}
	}
	return chunks
}

func (w *World) wantedChunks() map[ChunkCoord]struct{} {
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
	return wanted
}

func (w *World) PlayerActiveChunks(radius int32) []*Chunk {
	seen := make(map[ChunkCoord]struct{})
	chunks := make([]*Chunk, 0, len(w.Players)*9)
	for _, pl := range w.Players {
		cx := WorldToChunkCoord(int32(pl.X))
		cz := WorldToChunkCoord(int32(pl.Z))
		for dx := -radius; dx <= radius; dx++ {
			for dz := -radius; dz <= radius; dz++ {
				coord := ChunkCoord{X: cx + dx, Z: cz + dz}
				if _, dup := seen[coord]; dup {
					continue
				}
				seen[coord] = struct{}{}
				if c, ok := w.chunks[coord]; ok {
					chunks = append(chunks, c)
				}
			}
		}
	}
	return chunks
}

func (w *World) PopUnusedChunks() map[ChunkCoord]*Chunk {
	wanted := w.wantedChunks()

	w.Mu.Lock()
	defer w.Mu.Unlock()

	var removed map[ChunkCoord]*Chunk
	for coord, ch := range w.chunks {
		if _, ok := wanted[coord]; !ok {
			if removed == nil {
				removed = make(map[ChunkCoord]*Chunk, 4)
			}
			removed[coord] = ch
			delete(w.chunks, coord)
		}
	}
	if len(removed) > 0 {
		log.Printf("Popping %d chunks", len(removed))
	}
	return removed
}

func (w *World) IsLoaded(x, z int32) bool {
	cx := WorldToChunkCoord(x)
	cz := WorldToChunkCoord(z)
	coord := ChunkCoord{X: cx, Z: cz}
	if _, ok := w.chunks[coord]; ok {
		return true
	}
	return false
}

func (w *World) SnapshotEntities() []Entity {
	snapshot := make([]Entity, 0, len(w.Entities))
	for _, e := range w.Entities {
		snapshot = append(snapshot, e)
	}
	return snapshot
}

func (w *World) Size() int64 {
	w.Mu.RLock()
	defer w.Mu.RUnlock()
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
}

type ChunkLogic struct {
	Growables    map[BlockKey]Growable
	DroppedItems map[int32]*DroppedItem
}

func NewChunkLogic() *ChunkLogic {
	return &ChunkLogic{
		Growables:    make(map[BlockKey]Growable),
		DroppedItems: make(map[int32]*DroppedItem),
	}
}

// World holds all loaded chunks and is the single source of truth for block state.
type World struct {
	//Mu         dlock.DebugRWMutex
	Mu          sync.RWMutex
	sessionMu   sync.Map
	blockQueue  map[[3]int32]QueueBlock
	chunks      map[ChunkCoord]*Chunk
	Tick        int64
	Players     map[int32]*player.Player
	Entities    map[int32]Entity
	EntityCount int32
	WorldType   WorldType
	Scheduler   BlockUpdateScheduler

	TickSpeed       int64
	Containers      Containers
	ChestPlacements ChestPlacement
	WorldDir        string
	CommitHash      string
	Seed            int64
	noise           *PerlinNoise

	broadcastRelativePosition func(w *World, c Entity, prevX, prevY, prevZ, nextX, nextY, nextZ float64, yaw byte)
	collectItem               func(itemId, collectorId int32) []byte
	sendSetSlot               func(connection net.Conn, windowId byte, slot int16, item inventory.Item)
	broadcastEntityVelocity   func(w *World, entityId int32, vx, vy, vz float64)
	broascastDespawn          func(w *World, id int32)
	broadcastTeleport         func(w *World, c Entity, cx, cy, cz float64, yaw byte)
	broadcastContainerData    func(w *World, windowId byte, itemType, itemValue int16)
	broadcastSetSlot          func(w *World, windowId byte, slot int16, item inventory.Item)
	broadcastMultiBlockChange func(w *World, chunkX, chunkZ int32, numOfBlocks uint16, blockCoords []uint16, blockTypes, metadata []byte)
	broadcastBlockChange      func(w *World, x, y, z int32, blockType, blockMeta byte)
	broadcastTime             func(w *World, tick int64)
	broadcastSpawnObject      func(w *World, eId int32, oType byte, x, y, z, oeId int32, velX, velY, velZ int16)
}

func (w *World) BroadcastTime(tick int64) {
	w.broadcastTime(w, tick)
}

func (w *World) BroadcastSpawnObject(eId int32, oType byte, x, y, z, oeId int32, velX, velY, velZ int16) {
	w.broadcastSpawnObject(w, eId, oType, x, y, z, oeId, velX, velY, velZ)
}

func (w *World) BroadcastBlockChange(x, y, z int32, blockType, blockMeta byte) {
	w.broadcastBlockChange(w, x, y, z, blockType, blockMeta)
}

func (w *World) BroadcastMultiBlockChange(chunkX, chunkZ int32, numOfBlocks uint16, blockCoords []uint16, blockTypes, metadata []byte) {
	w.broadcastMultiBlockChange(w, chunkX, chunkZ, numOfBlocks, blockCoords, blockTypes, metadata)
}

func (w *World) BroadcastContainerData(windowId byte, itemType, itemValue int16) {
	w.broadcastContainerData(w, windowId, itemType, itemValue)
}

func (w *World) BroadcastSetSlot(windowId byte, slot int16, item inventory.Item) {
	w.broadcastSetSlot(w, windowId, slot, item)
}

func (w *World) BroadcastTeleport(c Entity, cx, cy, cz float64, yaw byte) {
	w.broadcastTeleport(w, c, cx, cy, cz, yaw)
}

func (w *World) BroadcastDespawn(id int32) {
	w.broascastDespawn(w, id)
}

func (w *World) BroadcastRelativePosition(c Entity, prevX, prevY, prevZ, nextX, nextY, nextZ float64, yaw byte) {
	w.broadcastRelativePosition(w, c, prevX, prevY, prevZ, nextX, nextY, nextZ, yaw)
}

func (w *World) BroadcastEntityVelocity(entityId int32, vx, vy, vz float64) {
	w.broadcastEntityVelocity(w, entityId, vx, vy, vz)
}

func (w *World) CollectItem(itemId, collectorId int32) []byte {
	return w.collectItem(itemId, collectorId)
}

func (w *World) SendSetSlot(connection net.Conn, windowId byte, slot int16, item inventory.Item) {
	w.sendSetSlot(connection, windowId, slot, item)
}

func (w *World) SetBroadcastRelativePosition(f func(w *World, c Entity, prevX, prevY, prevZ, nextX, nextY, nextZ float64, yaw byte)) {
	w.broadcastRelativePosition = f
}

func (w *World) SetCollectItem(f func(itemId, collectorId int32) []byte) {
	w.collectItem = f
}

func (w *World) SetSendSetSlot(f func(connection net.Conn, windowId byte, slot int16, item inventory.Item)) {
	w.sendSetSlot = f
}

func (w *World) SetBroadcastEntityVelocity(f func(w *World, entityId int32, vx, vy, vz float64)) {
	w.broadcastEntityVelocity = f
}

func (w *World) SetBroadcastDespawn(f func(world *World, id int32)) {
	w.broascastDespawn = f
}

func (w *World) SetBroadcastTeleport(f func(w *World, c Entity, cx, cy, cz float64, yaw byte)) {
	w.broadcastTeleport = f
}

func (w *World) SetBroadcastContainerData(f func(w *World, windowId byte, itemType, itemValue int16)) {
	w.broadcastContainerData = f
}

func (w *World) SetBroadcastSetSlot(f func(w *World, windowId byte, slot int16, item inventory.Item)) {
	w.broadcastSetSlot = f
}

func (w *World) SetBroadcastBlockChange(f func(w *World, x, y, z int32, blockType, blockMeta byte)) {
	w.broadcastBlockChange = f
}

func (w *World) SetBroadcastMultiBlockChange(f func(world *World, chunkX, chunkZ int32, numOfBlocks uint16, blockCoords []uint16, blockTypes, metadata []byte)) {
	w.broadcastMultiBlockChange = f
}

func (w *World) SetBroadcastTime(f func(w *World, tick int64)) {
	w.broadcastTime = f
}

func (w *World) SetBroadcastSpawnObject(f func(w *World, eId int32, oType byte, x, y, z, oeId int32, velX, velY, velZ int16)) {
	w.broadcastSpawnObject = f
}

func (w *World) LockSession(username string) func() {
	muIface, _ := w.sessionMu.LoadOrStore(username, &sync.Mutex{})
	mu := muIface.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func NewWorld(commitHash string, seed int64, worldType WorldType) *World {
	return &World{
		//Mu:          *dlock.NewDebugRWMutex("World"),
		Seed:        seed,
		noise:       NewPerlinNoise(seed),
		CommitHash:  commitHash,
		WorldDir:    "saves",
		WorldType:   worldType,
		chunks:      make(map[ChunkCoord]*Chunk),
		EntityCount: 0,
		Players:     make(map[int32]*player.Player),
		Entities:    make(map[int32]Entity),
		TickSpeed:   1,
		Containers: Containers{
			Chests:     make(map[BlockKey]*inventory.Chest),
			Dispensers: make(map[BlockKey]*inventory.Dispenser),
			Furnaces:   make(map[BlockKey]*inventory.Furnace),
		},
		ChestPlacements: ChestPlacement{
			AdjacentSlots:  make(map[BlockKey]BlockKey),
			ForbiddenSlots: make(map[BlockKey]struct{}),
		},
		Scheduler:  NewBlockUpdateScheduler(),
		blockQueue: make(map[[3]int32]QueueBlock),
	}
}

func (w *World) DebugEntities() {
	for i, e := range w.Entities {
		x, y, z := e.GetPosition()
		log.Printf("%d : x=%f, y=%f, z=%f", i, math.Round(x), math.Round(y), math.Round(z))
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
	entityId := w.NextEntityId()
	cx := WorldToChunkCoord(x)
	cz := WorldToChunkCoord(z)
	chunk := w.GetOrCreateChunk(cx, cz, w.WorldType)
	logic := chunk.Logic
	logic.DroppedItems[entityId] = &DroppedItem{EntityId: entityId, ItemId: itemId, Amount: amount, Metadata: meta, X: x, Y: y, Z: z, PickupDelay: pickupDelay}
	return entityId
}

func (w *World) RemoveDroppedItem(entityId int32, x, z int32) {
	cx := WorldToChunkCoord(x)
	cz := WorldToChunkCoord(z)
	chunk, ok := w.chunks[ChunkCoord{X: cx, Z: cz}]
	if ok {
		delete(chunk.Logic.DroppedItems, entityId)
	}
}

func (w *World) LoadChunks() []*Chunk {
	w.Mu.RLock()
	defer w.Mu.RUnlock()
	chunks := make([]*Chunk, 0, len(w.chunks))
	for _, c := range w.chunks {
		chunks = append(chunks, c)
	}
	return chunks
}

// ChunkExists reports whether the chunk at (cx, cz) has already been loaded/generated.
func (w *World) ChunkExists(cx, cz int32) bool {
	_, ok := w.chunks[ChunkCoord{cx, cz}]
	return ok
}

func printCallStack() {
	const depth = 32

	pcs := make([]uintptr, depth)
	n := runtime.Callers(2, pcs)

	frames := runtime.CallersFrames(pcs[:n])

	var callers []string

	for {
		frame, more := frames.Next()

		name := frame.Function

		if idx := strings.LastIndex(name, "."); idx != -1 {
			name = name[idx+1:]
		}

		callers = append(callers, name)

		if !more {
			break
		}
	}

	log.Printf("%s", strings.Join(callers, " <-- "))
}

// GetOrCreateChunk returns the chunk at (cx, cz), generating it if it doesn't exist yet.
func (w *World) GetOrCreateChunk(cx, cz int32, worldType WorldType) *Chunk {
	key := ChunkCoord{cx, cz}
	ch, ok := w.chunks[key]
	if ok {
		return ch
	}
	if w.WorldDir != "" {
		rx, rz := cx>>5, cz>>5
		lx, lz := cx&31, cz&31
		regionPath := filepath.Join(w.WorldDir, "region", mcregion.RegionFileName(rx*32, rz*32))

		lvl, err := mcregion.ReadChunk(regionPath, lx, lz)
		if err != nil {
			log.Printf("chunk (%d,%d): read failed, regenerating: %v", cx, cz, err)
		} else if lvl != nil {
			c, err := w.readChunkFromNBT(lvl, cx, cz)
			if err != nil {
				log.Printf("chunk (%d,%d): decode failed, regenerating: %v", cx, cz, err)
			} else {
				w.chunks[key] = c
				return c
			}
		}
	}
	c := w.generateChunk(cx, cz, w.WorldType)
	c.X = cx * CHUNK_SIZE_X
	c.Z = cz * CHUNK_SIZE_Z

	w.chunks[key] = c
	return c
}

// SetBlock updates a single block in the world using world-space coordinates.
func (w *World) SetBlock(worldX int32, worldY byte, worldZ int32, block Block) {
	cx := WorldToChunkCoord(worldX)
	cz := WorldToChunkCoord(worldZ)
	chunk := w.GetOrCreateChunk(cx, cz, w.WorldType)
	lx := WorldToLocalCoord(worldX)
	lz := WorldToLocalCoord(worldZ)
	chunk.SetBlock(lx, int(worldY), lz, block)

	key := BlockKey{worldX, worldY, worldZ}
	w.SetGrowable(block, key)
}

func (w *World) GetBlock(worldX int32, worldY byte, worldZ int32) Block {
	cx := WorldToChunkCoord(worldX)
	cz := WorldToChunkCoord(worldZ)
	chunk := w.GetOrCreateChunk(cx, cz, w.WorldType)

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
	w.Entities[int32(entityId)] = &r
}

func (w *World) AddEntity(e Entity) {
	w.Entities[e.GetEntityId()] = e
}

func (w *World) RemovePlayer(p *player.Player) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	delete(w.Players, int32(p.EntityId))
	delete(w.Entities, int32(p.EntityId))
}

func (w *World) RemoveEntity(entityId int32) {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	delete(w.Entities, entityId)
}

func (w *World) AdvanceTime() {
	w.BroadcastTime(w.Tick)
}

// BroadcastPacket sends raw pre-serialized packet data to all logged-in players.
func (w *World) BroadcastPacket(data []byte) {
	for _, pl := range w.Players {
		if pl.LoggedIn {
			pl.Connection.Write(data)
		}
	}
}

func (w *World) MulticastPacket(data []byte, exclude *player.Player) {
	for _, pl := range w.Players {
		if pl.LoggedIn && pl != exclude {
			pl.Connection.Write(data)
		}
	}
}

func (w *World) ForEachPlayer(fn func(*player.Player)) {
	for _, pl := range w.Players {
		if pl.LoggedIn {
			fn(pl)
		}
	}
}

type QueueBlock struct {
	X        int32
	Y        byte
	Z        int32
	TypeID   byte
	Metadata byte
}

func (w *World) SetBlockInQueue(x, y, z int32, block Block) {
	w.SetBlock(x, byte(y), z, block)

	if w.blockQueue == nil {
		w.blockQueue = make(map[[3]int32]QueueBlock)
	}

	w.blockQueue[[3]int32{x, y, z}] = QueueBlock{
		X:        x,
		Y:        byte(y),
		Z:        z,
		TypeID:   block.TypeId,
		Metadata: block.Metadata,
	}
}

func (w *World) FlushBlockQueue() {
	blocks := w.blockQueue
	w.blockQueue = nil
	if len(blocks) <= 10 {
		for _, b := range blocks {
			w.BroadcastBlockChange(b.X, int32(b.Y), b.Z, b.TypeID, b.Metadata)
		}
		return
	}

	type chunkChange struct {
		coords []uint16
		types  []byte
		meta   []byte
	}

	changes := make(map[[2]int32]*chunkChange)

	for _, b := range blocks {
		chunkX := WorldToChunkCoord(b.X)
		chunkZ := WorldToChunkCoord(b.Z)

		key := [2]int32{chunkX, chunkZ}

		change, ok := changes[key]
		if !ok {
			change = &chunkChange{}
			changes[key] = change
		}

		coord := uint16((b.X&15)<<12 |
			(b.Z&15)<<8 |
			int32(b.Y))

		change.coords = append(change.coords, coord)
		change.types = append(change.types, b.TypeID)
		change.meta = append(change.meta, b.Metadata)
	}

	for chunk, change := range changes {
		w.BroadcastMultiBlockChange(chunk[0], chunk[1], uint16(len(change.coords)), change.coords, change.types, change.meta)
	}
}
