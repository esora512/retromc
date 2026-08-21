package level

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/player"
)

const VIEW_DISTANCE = 12

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

func (w *World) GetPlayerByUsername(name string) (*player.Player, bool) {
	for _, pl := range w.Players {
		if pl.Username == name {
			return pl, true
		}
	}
	return nil, false
}

func (w *World) IsNight() bool {
	timeTicks := w.TimeTick % 24000
	return timeTicks >= 12541 && timeTicks < 23458
}


func (w *World) SnapshotEntities() []constants.Entity {
	snapshot := make([]constants.Entity, 0, len(w.Entities))
	for _, e := range w.Entities {
		snapshot = append(snapshot, e)
	}
	return snapshot
}

// World holds all loaded chunks and is the single source of truth for block state.
type World struct {
	//Mu         dlock.DebugRWMutex
	Mu             sync.RWMutex
	Rand           *rand.Rand
	chunkLoadGroup singleflight.Group
	sessionMu      sync.Map
	blockQueue     map[[4]int32]QueueBlock
	oChunks        map[ChunkCoord]*Chunk
	nChunks        map[ChunkCoord]*Chunk
	Tick           int64
	TimeTick       int64
	Players        map[int32]*player.Player
	Entities       map[int32]constants.Entity
	EntityCount    int32
	WorldType      WorldType
	Scheduler      BlockUpdateScheduler

	OppedUsernames map[string]bool

	TickSpeed       int64
	Containers      Containers
	ChestPlacements ChestPlacement
	WorldDir        string
	CommitHash      string
	Seed            int64
	sleepers        map[int32]int

	broadcastPositionAndRotation    func(w *World, c constants.Entity, prevX, prevY, prevZ, nextX, nextY, nextZ float64, yaw byte)
	collectItem                     func(itemId, collectorId int32) []byte
	sendSetSlot                     func(connection net.Conn, windowId byte, slot int16, item inventory.Item)
	broadcastEntityVelocity         func(w *World, entityId int32, vx, vy, vz float64)
	broascastDespawn                func(w *World, id int32)
	broadcastTeleport               func(w *World, c constants.Entity, cx, cy, cz float64, yaw byte)
	broadcastContainerData          func(w *World, windowId byte, itemType, itemValue int16)
	broadcastSetSlot                func(w *World, windowId byte, slot int16, item inventory.Item)
	broadcastMultiBlockChange       func(w *World, chunkX, chunkZ int32, numOfBlocks uint16, blockCoords []uint16, blockTypes, metadata []byte)
	broadcastBlockChange            func(w *World, x, y, z int32, blockType, blockMeta byte)
	broadcastTime                   func(w *World, tick int64)
	broadcastSpawnObject            func(w *World, eId int32, oType byte, x, y, z, oeId int32, velX, velY, velZ int16)
	broadcastWakeUp                 func(w *World, id int32)
	broadcastWorldMsg               func(w *World, msg string)
	broadcastMobSpawn               func(w *World, mobType, meta byte, x, y, z int32, yaw, pitch byte, dim int32, entityId int32)
	broadcastMobPositionAndRotation func(w *World, m *entities.Mob, nX, nY, nZ, yaw, pitch float64)
	newMobPositionAndRotationPacket func(m *entities.Mob, nX, nY, nZ, yaw, pitch float64) []byte
	newEntityVelocityPacket         func(entityId int32, m constants.MovementState) []byte
	createAndSetMovementDroppedItem func(world *World, x, y, z float64, blockItem int16, blockMeta byte, count byte, dim, delay int32)

	sendSetHealth func(conn net.Conn, hp uint16)
	broadcastPain func(w *World, entityId int32)

	newPositionAndRotationOrTeleportPacket func(w *World, e constants.Entity, m constants.MovementState) []byte
	newTeleportPacket func(w *World, e constants.Entity, m constants.MovementState) []byte
	newRotationPacket func(w *World, e constants.Entity, m constants.MovementState) []byte
	newPositionPacket func(w *World, e constants.Entity, m constants.MovementState) []byte
	newMobPositionAndRotationOrTeleportPacket func(e constants.Entity, m constants.MovementState) []byte

	spawnPlayer func(pl *player.Player) []byte
	spawnObject func(e constants.Entity) []byte
	spawnMob func(m *entities.Mob) []byte
	spawnItem func(d *DroppedItem) []byte
	despawnEntity func(id int32) []byte
	setEquipment func(pl *player.Player, send func([]byte) (int, error))
}

func (w *World) SetSpawnPlayer(f func(pl *player.Player) []byte) {
	w.spawnPlayer = f
}

func (w *World) SetSpawnObject(f func(e constants.Entity) []byte) {
	w.spawnObject = f
}

func (w *World) SetSpawnMob(f func(m *entities.Mob) []byte) {
	w.spawnMob = f
}

func (w *World) SetSpawnItem(f func(m *DroppedItem) []byte) {
	w.spawnItem = f
}

func (w *World) SetDespawnEntity(f func(id int32) []byte) {
	w.despawnEntity = f
}

func (w *World) SetSendEquipment(f func(pl *player.Player, send func([]byte) (int, error))) {
	w.setEquipment = f 
}

func (w *World) SetNewMobPositionAndRotationOrTeleportPacket(f func(e constants.Entity, m constants.MovementState) []byte) {
	w.newMobPositionAndRotationOrTeleportPacket = f
}

func (w *World) SetNewRotationPacket(f func(w *World, e constants.Entity, m constants.MovementState) []byte) {
	w.newPositionPacket = f
}

func (w *World) SetNewPositionPacket(f func(w *World, e constants.Entity, m constants.MovementState) []byte) {
	w.newRotationPacket = f
}

func (w *World) SetNewTeleportPacket(f func(w *World, e constants.Entity, m constants.MovementState) []byte) {
	w.newTeleportPacket = f
}

func (w *World) SetNewPositionAndRotationOrTeleportPacket(f func(w *World, e constants.Entity, m constants.MovementState) []byte) {
	w.newPositionAndRotationOrTeleportPacket = f
}

func (w *World) SetAndCreateAndSetMovementDroppedItem(f func(world *World, x, y, z float64, blockItem int16, blockMeta byte, count byte, dim, delay int32)) {
	w.createAndSetMovementDroppedItem = f
}

func (w *World) CreateAndSetMovementDroppedItem(x, y, z float64, blockItem int16, blockMeta byte, count byte, dim, delay int32) {
	w.createAndSetMovementDroppedItem(w, x, y, z, blockItem, blockMeta, count, dim, delay)
}

func (w *World) SetNewEntityVelocityPacket(f func(entityId int32, m constants.MovementState) []byte) {
	w.newEntityVelocityPacket = f
}

func (w *World) SetNewMobPositionAndRotationPacket(f func(m *entities.Mob, nX, nY, nZ, yaw, pitch float64) []byte) {
	w.newMobPositionAndRotationPacket = f
}

func (w *World) BroadcastPain(entityId int32) {
	w.broadcastPain(w, entityId)
}

func (w *World) SendSetHealth(conn net.Conn, health uint16) {
	w.sendSetHealth(conn, health)
}

func (w *World) BroadcastMobPositionAndRotation(m *entities.Mob, nX, nY, nZ, yaw, pitch float64) {
	w.broadcastMobPositionAndRotation(w, m, nX, nY, nZ, yaw, pitch)
}

func (w *World) BroadcastMobSpawn(mobType, meta byte, x, y, z int32, yaw, pitch byte, dim int32, entityId int32) {
	w.broadcastMobSpawn(w, mobType, meta, x, y, z, yaw, pitch, dim, entityId)
}

func (w *World) GetEntity(id int32) (constants.Entity, bool) {
	if e, ok := w.Entities[id]; ok {
		return e, ok
	}
	return nil, false
}

func (w *World) BroadcastWorldMsg(msg string) {
	w.broadcastWorldMsg(w, msg)
}

func (w *World) BroadcastWakeUp(id int32) {
	w.broadcastWakeUp(w, id)
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

func (w *World) BroadcastTeleport(c constants.Entity, cx, cy, cz float64, yaw byte) {
	w.broadcastTeleport(w, c, cx, cy, cz, yaw)
}

func (w *World) BroadcastDespawn(id int32) {
	w.broascastDespawn(w, id)
}

func (w *World) BroadcastPositionAndRotation(c constants.Entity, prevX, prevY, prevZ, nextX, nextY, nextZ float64, yaw byte) {
	w.broadcastPositionAndRotation(w, c, prevX, prevY, prevZ, nextX, nextY, nextZ, yaw)
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

func (w *World) SetBroadcastPositionAndRotation(f func(w *World, c constants.Entity, prevX, prevY, prevZ, nextX, nextY, nextZ float64, yaw byte)) {
	w.broadcastPositionAndRotation = f
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

func (w *World) SetBroadcastTeleport(f func(w *World, c constants.Entity, cx, cy, cz float64, yaw byte)) {
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

func (w *World) SetBroadcastWakeUp(f func(w *World, id int32)) {
	w.broadcastWakeUp = f
}

func (w *World) SetBroadcastWorldMsg(f func(w *World, msg string)) {
	w.broadcastWorldMsg = f
}

func (w *World) SetBroadcastMobSpawn(f func(w *World, mobType, meta byte, x, y, z int32, yaw, pitch byte, dim int32, entityId int32)) {
	w.broadcastMobSpawn = f
}

func (w *World) SetBroadcastMobPositionAndRotation(f func(w *World, m *entities.Mob, nX, nY, nZ, yaw, pitch float64)) {
	w.broadcastMobPositionAndRotation = f
}

func (w *World) SetSendSetHealth(f func(connection net.Conn, health uint16)) {
	w.sendSetHealth = f
}

func (w *World) SetBroadcastPain(f func(w *World, entityId int32)) {
	w.broadcastPain = f
}

func (w *World) SetOppedUsernames(names map[string]bool) {
	w.OppedUsernames = names
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
		CommitHash:  commitHash,
		WorldDir:    "saves",
		WorldType:   worldType,
		oChunks:     make(map[ChunkCoord]*Chunk),
		nChunks:     make(map[ChunkCoord]*Chunk),
		EntityCount: 0,
		Players:     make(map[int32]*player.Player),
		Entities:    make(map[int32]constants.Entity),
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
		blockQueue: make(map[[4]int32]QueueBlock),
		sleepers:   make(map[int32]int),
		Rand:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (w *World) AddSleeper(pl *player.Player) {
	w.sleepers[pl.GetEntityId()] = 0
}

func (w *World) RemoveSleeper(pl *player.Player) {
	delete(w.sleepers, pl.GetEntityId())
}

func (w *World) Sleep() {
	for k := range w.sleepers {
		w.sleepers[k] += 1
	}
}

func (w *World) SleepThroughNight() {
	var wokeUp bool
	for k, s := range w.sleepers {
		if s > 30 {
			toNextDay := 24000 - (w.TimeTick % 24000)
			w.TimeTick += toNextDay
			w.BroadcastWakeUp(k)
			wokeUp = true
			w.BroadcastWorldMsg("Sleeping through the night...")
			break
		}
	}
	if wokeUp {
		clear(w.sleepers)
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

func (w *World) AddDroppedItem(x, y, z float64, itemId int32, amount, meta byte, pickupDelay, dim int32, velX, velY, velZ float64) int32 {
	entityId := w.NextEntityId()
	w.Mu.Lock()
	defer w.Mu.Unlock()
	w.Entities[entityId] = &DroppedItem{EntityId: entityId, ItemId: itemId, Amount: amount, Metadata: meta, X: x, Y: y, Z: z, PickupDelay: pickupDelay, VelX: velX, VelY: velY, VelZ: velZ}
	return entityId
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

// SetBlock updates a single block in the world using world-space coordinates.
func (w *World) SetBlock(worldX int32, worldY byte, worldZ int32, block constants.WBlock, dim int32) {
	cx := WorldToChunkCoord(worldX)
	cz := WorldToChunkCoord(worldZ)
	chunk := w.GetOrCreateChunk(cx, cz, dim)
	chunk.HasChanged = true
	lx := WorldToLocalCoord(worldX)
	lz := WorldToLocalCoord(worldZ)
	chunk.SetBlock(lx, int(worldY), lz, block)

	key := BlockKey{worldX, worldY, worldZ}
	w.SetGrowable(block, key, dim)
}

func (w *World) GetBlock(worldX int32, worldY byte, worldZ int32, dim int32) constants.WBlock {
	cx := WorldToChunkCoord(worldX)
	cz := WorldToChunkCoord(worldZ)
	chunk := w.GetOrCreateChunk(cx, cz, dim)

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

func (w *World) AddRidable(entityId, ownerEntityId int32, x, y, z, vx, vy, vz float64, objectType byte, dim int32) {
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
		Dimension:     dim,
	}
	w.Entities[int32(entityId)] = &r
}

func (w *World) AddEntity(e constants.Entity) {
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
	w.TimeTick += 1
	w.BroadcastTime(w.TimeTick)
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
	Dim      int32
}

func (w *World) SetBlockInQueue(x, y, z int32, block constants.WBlock, dim int32) {
	w.SetBlock(x, byte(y), z, block, dim)

	if w.blockQueue == nil {
		w.blockQueue = make(map[[4]int32]QueueBlock)
	}

	w.blockQueue[[4]int32{x, y, z, dim}] = QueueBlock{
		X:        x,
		Y:        byte(y),
		Z:        z,
		TypeID:   block.TypeId,
		Metadata: block.Metadata,
		Dim:      dim,
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
