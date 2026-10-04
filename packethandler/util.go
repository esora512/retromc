package packethandler

import (
	"fmt"
	"net"

	"math"
	"sort"
	"time"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

// SendSetSlot tells the client to update a single inventory slot.
func SendSetSlot(connection net.Conn, windowId byte, slot int16, item inventory.Item) {
	setSlotPacket := packets.SetSlotPacket{
		WindowId: windowId,
		Slot:     slot,
		Item:     item,
	}
	connection.Write(setSlotPacket.Serialize())
}

func sendChestContents(connection net.Conn, chest *inventory.Chest) {
	for i := int16(0); i < int16(chest.Size); i++ {
		item := chest.PeekItem(i)
		if item.TypeId != -1 {
			SendSetSlot(connection, 1, i, item)
		}
	}
}

func sendDispenserContents(connection net.Conn, dispenser *inventory.Dispenser) {
	for i := int16(0); i < int16(dispenser.Size); i++ {
		item := dispenser.PeekItem(i)
		if item.TypeId != -1 {
			SendSetSlot(connection, 1, i, item)
		}
	}
}

func sendFurnaceContents(connection net.Conn, furnace *inventory.Furnace) {
	for i := int16(0); i < int16(furnace.Size); i++ {
		item := furnace.PeekItem(i)
		if item.TypeId != -1 {
			SendSetSlot(connection, 1, i, item)
		}
	}
}

func broadcastChestContents(world *level.World, source *player.Player, chest *inventory.Chest) {
	world.ForEachPlayer(func(pl *player.Player) {
		if pl == source || pl.InventoryType != player.ChestInventory {
			return
		}
		if world.GetChest(pl.Chest.X, pl.Chest.Y, pl.Chest.Z, pl.Chest.Dim) == chest {
			for i := int16(0); i < int16(chest.Size); i++ {
				SendSetSlot(pl.Connection, 1, i, chest.PeekItem(i))
			}
		}
	})
}

func broadcastDispenserContents(world *level.World, source *player.Player, dispenser *inventory.Dispenser) {
	world.ForEachPlayer(func(pl *player.Player) {
		if pl == source || pl.InventoryType != player.DispenserInventory {
			return
		}
		if world.GetDispenser(pl.Dispenser.X, pl.Dispenser.Y, pl.Dispenser.Z, pl.Dispenser.Dim) == dispenser {
			for i := int16(0); i < int16(dispenser.Size); i++ {
				SendSetSlot(pl.Connection, 1, i, dispenser.PeekItem(i))
			}
		}
	})
}

func broadcastFurnaceContents(world *level.World, source *player.Player, furnace *inventory.Furnace) {
	world.ForEachPlayer(func(pl *player.Player) {
		if pl == source || pl.InventoryType != player.FurnaceInventory {
			return
		}
		if world.GetFurnace(pl.Furnace.X, pl.Furnace.Y, pl.Furnace.Z, pl.Furnace.Dim) == furnace {
			for i := int16(0); i < int16(furnace.Size); i++ {
				SendSetSlot(pl.Connection, 1, i, furnace.PeekItem(i))
			}
		}
	})
}

// // presetInventory writes the starting items directly into the player's in-memory
// // inventory. The caller is responsible for sending the inventory to the client.
// func presetInventory(inv *inventory.Inventory) {
// 	return
// 	// inv.SetItem(36, constants.Rail.Value, 16, 0)
// 	// inv.SetItem(37, constants.Minecart.Value, 1, 0)
// 	// inv.SetItem(38, constants.Stone.Value, 64, 0)
// 	// inv.SetItem(39, constants.DiamondPickaxe.Value, 1, 0)
// }

// Spiral order around the player, so the closest chunks are sent first.
var viewOffsets = spiralOffsets(level.VIEW_DISTANCE)

func inViewDistance(coord player.ChunkCoord, cx, cz int32) bool {
	dx, dz := coord.X-cx, coord.Z-cz
	return dx >= -level.VIEW_DISTANCE && dx <= level.VIEW_DISTANCE && dz >= -level.VIEW_DISTANCE && dz <= level.VIEW_DISTANCE
}

// unloadFarChunks tells the client to drop the chunks that fell out of range.
func unloadFarChunks(pl *player.Player, cx, cz int32) {
	for coord := range pl.PendingChunks {
		if !inViewDistance(coord, cx, cz) {
			delete(pl.PendingChunks, coord)
		}
	}
	for coord := range pl.SentChunks {
		if !inViewDistance(coord, cx, cz) {
			unload := packets.SetChunkVisibilityPacket{X: coord.X, Z: coord.Z, Mode: false}
			pl.Connection.Write(unload.Serialize())
			delete(pl.SentChunks, coord)
		}
	}
}

func initialUpdateChunks(world *level.World, x, z float64, pl *player.Player, onComplete func()) {
	cx := level.WorldToChunkCoord(int32(x))
	cz := level.WorldToChunkCoord(int32(z))

	// Nothing to do if we haven't crossed a chunk boundary
	if cx == pl.LastChunkX && cz == pl.LastChunkZ && pl.HasInitializedChunks {
		onComplete()
		return
	}

	dim := pl.Dimension
	var pending []level.ChunkCoord

	for _, off := range viewOffsets {
		if abs32(off.X) > level.SpawnChunkRadius || abs32(off.Z) > level.SpawnChunkRadius {
			continue
		}
		coord := level.ChunkCoord{X: cx + off.X, Z: cz + off.Z}

		if pl.SentChunks.Has(coord.X, coord.Z) {
			continue
		}

		if chunk, ok := world.PeekChunk(coord.X, coord.Z, dim); ok {
			sendChunkToPlayer(world, pl, coord, dim, chunk)
			continue
		}

		pending = append(pending, coord)
	}

	unloadFarChunks(pl, cx, cz)
	pl.LastChunkX = cx
	pl.LastChunkZ = cz

	finish := func() {
		pl.HasInitializedChunks = true
		applyChunkVisibility(world, pl, cx, cz)
		pl.LastDim = dim
		onComplete()
	}

	if len(pending) == 0 {
		finish()
		return
	}

	remaining := len(pending)
	for _, coord := range pending {
		world.RequestChunkAsync(coord.X, coord.Z, dim, func(generated *level.Chunk) {
			world.Enqueue(func() {
				if !world.HasPlayer(pl) {
					return // player disconnected while this chunk was generating
				}
				chunk := world.InsertChunk(coord.X, coord.Z, dim, generated)
				if !pl.SentChunks.Has(coord.X, coord.Z) {
					sendChunkToPlayer(world, pl, coord, dim, chunk)
				}
				remaining--
				if remaining == 0 {
					finish()
				}
			})
		})
	}
}

// sendChunkToPlayer must run on the game loop, since lighting reads the neighbouring chunks.
func sendChunkToPlayer(world *level.World, pl *player.Player, coord level.ChunkCoord, dim int32, chunk *level.Chunk) int {
	visibility := packets.SetChunkVisibilityPacket{X: coord.X, Z: coord.Z, Mode: true}
	pre := visibility.Serialize()
	pl.Connection.Write(pre)

	light := world.ComputeLight(coord.X, coord.Z, dim, chunk)
	data := packets.NewChunkBlockRegionPacket(chunk, light)
	player.WriteOwned(pl.Connection, data)
	light.Release()

	pl.SentChunks.Set(coord.X, coord.Z)
	return len(pre) + len(data)
}

func WorldToLocalCoord(world int32) int {
	return int(world & 15)
}

func spiralOffsets(radius int32) []level.ChunkCoord {
	size := 2*radius + 1
	total := size * size
	offsets := make([]level.ChunkCoord, 0, total)

	var x, z int32 = 0, 0
	var dx, dz int32 = 0, -1

	for i := int32(0); i < total*4; i++ { // upper bound; we break once we have enough
		if x >= -radius && x <= radius && z >= -radius && z <= radius {
			offsets = append(offsets, level.ChunkCoord{X: x, Z: z})
			if int32(len(offsets)) == total {
				break
			}
		}
		if x == z || (x < 0 && x == -z) || (x > 0 && x == 1-z) {
			dx, dz = -dz, dx
		}
		x, z = x+dx, z+dz
	}

	return offsets
}

const (
	chunkMoveThreshold   = level.CHUNK_SIZE_X / 2
	chunkMoveThresholdSq = chunkMoveThreshold * chunkMoveThreshold
)

func updateChunks(world *level.World, x, z float64, pl *player.Player) {
	// //NOTE: Important for debugging of how much this is halting.
	// start := time.Now()
	// defer func() {
	// 	log.Printf("UpdateChunks total: %s", time.Since(start))
	// }()

	// dx := x - pl.LastUpdateX
	// dz := z - pl.LastUpdateZ
	// // if dx*dx+dz*dz < chunkMoveThresholdSq {
	// // 	return
	// // }

	cx := level.WorldToChunkCoord(int32(x))
	cz := level.WorldToChunkCoord(int32(z))

	pl.LastUpdateX = x
	pl.LastUpdateZ = z

	if cx == pl.LastChunkX && cz == pl.LastChunkZ && pl.Dimension == pl.LastDim {
		return
	}

	// if pl.HasInitializedChunks {
	// 	// If initial chunks are visible, generate the rest later on as the player is already in the world
	// 	world.Enqueue(func() { applyChunkVisibility(world, pl, cx, cz) })
	// } else {
	// 	applyChunkVisibility(world, pl, cx, cz)
	// }

	applyChunkVisibility(world, pl, cx, cz)

	pl.LastChunkX = cx
	pl.LastChunkZ = cz
	pl.LastDim = pl.Dimension
}

func applyChunkVisibility(world *level.World, pl *player.Player, cx, cz int32) {
	pl.SentChunksMu.Lock()
	defer pl.SentChunksMu.Unlock()

	for _, off := range viewOffsets {
		coord := level.ChunkCoord{X: cx + off.X, Z: cz + off.Z}

		if pl.SentChunks.Has(coord.X, coord.Z) {
			continue
		}
		if pl.PendingChunks == nil {
			pl.PendingChunks = make(player.ChunkSet)
		}
		pl.PendingChunks.Set(coord.X, coord.Z)
	}

	unloadFarChunks(pl, cx, cz)
}

const (
	maxChunksPerTick = 16
	maxChunkBacklog  = 8
	chunkWindowBytes = 192 * 1024
	chunkAckTimeout  = 10 * time.Second
)

func FlushPendingChunks(world *level.World) {
	world.ForEachPlayer(func(pl *player.Player) {
		if len(pl.PendingChunks) == 0 {
			return
		}
		pl.SentChunksMu.Lock()
		defer pl.SentChunksMu.Unlock()

		coords := make([]player.ChunkCoord, 0, len(pl.PendingChunks))
		for c := range pl.PendingChunks {
			coords = append(coords, c)
		}
		dist := func(c player.ChunkCoord) int32 {
			dx, dz := c.X-pl.LastChunkX, c.Z-pl.LastChunkZ
			return dx*dx + dz*dz
		}
		sort.Slice(coords, func(i, j int) bool { return dist(coords[i]) < dist(coords[j]) })

		expireChunkAcks(pl)
		sent, bytes := 0, 0
		defer func() {
			if bytes > 0 {
				sendChunkAck(pl, bytes)
			}
		}()
		for _, c := range coords {
			if sent >= maxChunksPerTick || player.Backlog(pl.Connection) >= maxChunkBacklog || pl.ChunkBytesInFlight+bytes >= chunkWindowBytes {
				return
			}
			coord := level.ChunkCoord{X: c.X, Z: c.Z}
			chunk, ok := world.PeekChunk(c.X, c.Z, pl.Dimension)
			if !ok {
				requestChunkForPlayer(world, pl, coord)
				continue
			}
			bytes += sendChunkToPlayer(world, pl, coord, pl.Dimension, chunk)
			delete(pl.PendingChunks, c)
			sent++
		}
	})
}

func sendChunkAck(pl *player.Player, bytes int) {
	pl.NextChunkAck--
	if pl.NextChunkAck >= 0 {
		pl.NextChunkAck = -1
	}
	ping := packets.ContainerTransactionPacket{WindowId: 0, ActionNumber: pl.NextChunkAck, Accepted: false}
	pl.Connection.Write(ping.Serialize())
	pl.ChunkAcks = append(pl.ChunkAcks, player.ChunkAck{Id: pl.NextChunkAck, Bytes: bytes, SentAt: time.Now()})
	pl.ChunkBytesInFlight += bytes
}

func acknowledgeChunks(pl *player.Player, id int16) {
	for i, ack := range pl.ChunkAcks {
		if ack.Id != id {
			continue
		}
		for _, done := range pl.ChunkAcks[:i+1] {
			pl.ChunkBytesInFlight -= done.Bytes
		}
		pl.ChunkAcks = pl.ChunkAcks[i+1:]
		return
	}
}

func expireChunkAcks(pl *player.Player) {
	for len(pl.ChunkAcks) > 0 && time.Since(pl.ChunkAcks[0].SentAt) > chunkAckTimeout {
		pl.ChunkBytesInFlight -= pl.ChunkAcks[0].Bytes
		pl.ChunkAcks = pl.ChunkAcks[1:]
	}
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func requestChunkForPlayer(world *level.World, pl *player.Player, coord level.ChunkCoord) {
	if pl.RequestedChunks == nil {
		pl.RequestedChunks = make(player.ChunkSet)
	}
	if pl.RequestedChunks.Has(coord.X, coord.Z) {
		return
	}
	pl.RequestedChunks.Set(coord.X, coord.Z)
	dim := pl.Dimension
	world.RequestChunkAsync(coord.X, coord.Z, dim, func(generated *level.Chunk) {
		world.Enqueue(func() {
			delete(pl.RequestedChunks, player.ChunkCoord{X: coord.X, Z: coord.Z})
			world.InsertChunk(coord.X, coord.Z, dim, generated)
		})
	})
}

func decodeChunkCoord(key string) (level.ChunkCoord, bool) {
	var x, z int32
	n, err := fmt.Sscanf(key, "%d,%d", &x, &z)
	if err != nil || n != 2 {
		return level.ChunkCoord{}, false
	}
	return level.ChunkCoord{X: x, Z: z}, true
}

// TODO: Crashes the client for some reason
// func SendSpawnPosition(connection net.Conn) {
// 	spawnPositionPacket := packets.SpawnPositionOutPacket{
// 		X: 0,
// 		Y: 64,
// 		Z: 0,
// 	}
// 	outData := spawnPositionPacket.Serialize()
// 	connection.Write(outData)
// }

func sendInventory(connection net.Conn, pl *player.Player, w *level.World) {
	pkt := packets.FillContainerPacket{
		WindowId: 0, // 0 = player inventory
		Count:    int16(pl.Inventory.Size),
		Payload:  pl.Inventory,
	}
	connection.Write(pkt.Serialize())
}

func sendPlayerPositionAndLook(connection net.Conn, x, z float64, y float64) {
	packet := packets.PlayerPositionAndRotationPacket{
		X:        x,
		Y:        y,
		Stance:   y + playerEyeHeight, // S->C sends this in the eye slot
		Z:        z,
		Yaw:      0,
		Pitch:    0,
		OnGround: true,
	}
	outData := packet.Serialize()
	connection.Write(outData)
}

func sendEquipmentChangeForHotbarSlot(world *level.World, pl *player.Player) {
	world.ForEachPlayer(func(other *player.Player) {
		if other == pl {
			return
		}
		packets.SetEquipment(pl, func(b []byte) {
			other.Connection.Write(b)
		})
	})
}

func BroadcastTeleportPlayer(w *level.World, c constants.Entity, cx, cy, cz float64, yaw byte) {
	tpkt := packets.TeleportEntityPacket{
		EntityId: c.GetEntityId(),
		X:        int32(math.Floor(cx * 32)),
		Y:        int32(math.Floor(cy * 32)),
		Z:        int32(math.Floor(cz * 32)),
		Yaw:      yaw,
		Pitch:    0,
	}
	data := tpkt.Serialize()
	// viewers got an absolute position, make the tracker resend from scratch
	c.GetMovementState().EncInit = false

	for _, pl := range w.Players {
		if !pl.LoggedIn {
			continue
		}
		if pl.GetEntityId() == c.GetEntityId() {
			selfPkt := packets.PlayerPositionAndRotationPacket{
				X: cx, Y: cy, Z: cz, Stance: cy + playerEyeHeight, OnGround: true,
				Yaw:   float32(yaw) * 360.0 / 256.0,
				Pitch: 0,
			}
			pl.Connection.Write(selfPkt.Serialize())
			continue
		}
		pl.Connection.Write(data)
	}
}

func NewTeleportPacket(e constants.Entity, m constants.MovementState) []byte {
	dYaw := int32(math.Floor(float64(m.Yaw) * 256 / 360))
	dPitch := int32(math.Floor(float64(m.Pitch) * 256 / 360))
	tpkt := packets.TeleportEntityPacket{
		EntityId: e.GetEntityId(),
		X:        int32(math.Floor(m.X * 32)),
		Y:        int32(math.Floor(m.Y * 32)),
		Z:        int32(math.Floor(m.Z * 32)),
		Yaw:      byte(dYaw),
		Pitch:    byte(dPitch),
	}
	return tpkt.Serialize()
}

func NewPositionOrTeleportPacket(e constants.Entity, m constants.MovementState) []byte {
	encPrevX := int32(math.Floor(m.PrevX * 32))
	encPrevY := int32(math.Floor(m.PrevY * 32))
	encPrevZ := int32(math.Floor(m.PrevZ * 32))
	encNextX := int32(math.Floor(m.X * 32))
	encNextY := int32(math.Floor(m.Y * 32))
	encNextZ := int32(math.Floor(m.Z * 32))
	dX := encNextX - encPrevX
	dY := encNextY - encPrevY
	dZ := encNextZ - encPrevZ

	dYaw := int32(math.Floor(float64(m.Yaw) * 256 / 360))
	dPitch := int32(math.Floor(float64(m.Pitch) * 256 / 360))

	if m.KeepRotation {
		dYaw = int32(m.Yaw)
		dPitch = int32(m.Pitch)
	}

	if dX < -128 || dX > 127 || dY < -128 || dY > 127 || dZ < -128 || dZ > 127 {
		tpkt := packets.TeleportEntityPacket{
			EntityId: e.GetEntityId(),
			X:        int32(math.Floor(m.X * 32)),
			Y:        int32(math.Floor(m.Y * 32)),
			Z:        int32(math.Floor(m.Z * 32)),
			Yaw:      byte(dYaw),
			Pitch:    byte(dPitch),
		}
		return tpkt.Serialize()
	}
	p := packets.EntityPositionPacket{
		EntityId: e.GetEntityId(),
		X:        byte(dX),
		Y:        byte(dY),
		Z:        byte(dZ),
	}
	return p.Serialize()
}

func NewAnimationPacket(playerId int32, animation byte) []byte {
	p := packets.AnimationPacket{
		PlayerId:  playerId,
		Animation: animation,
	}
	return p.Serialize()
}

func NewRotationPacket(e constants.Entity, m constants.MovementState) []byte {
	dYaw := int32(math.Floor(float64(m.Yaw) * 256 / 360))
	dPitch := int32(math.Floor(float64(m.Pitch) * 256 / 360))

	if m.KeepRotation {
		dYaw = int32(m.Yaw)
		dPitch = int32(m.Pitch)
	}

	p := packets.EntityRotationPacket{
		EntityId: e.GetEntityId(),
		Yaw:      byte(dYaw),
		Pitch:    byte(dPitch),
	}
	return p.Serialize()
}

func NewPositionAndRotationOrTeleportPacket(e constants.Entity, m constants.MovementState) []byte {
	encPrevX := int32(math.Floor(m.PrevX * 32))
	encPrevY := int32(math.Floor(m.PrevY * 32))
	encPrevZ := int32(math.Floor(m.PrevZ * 32))
	encNextX := int32(math.Floor(m.X * 32))
	encNextY := int32(math.Floor(m.Y * 32))
	encNextZ := int32(math.Floor(m.Z * 32))
	dX := encNextX - encPrevX
	dY := encNextY - encPrevY
	dZ := encNextZ - encPrevZ
	dYaw := int32(math.Floor(float64(m.Yaw) * 256 / 360))
	dPitch := int32(math.Floor(float64(m.Pitch) * 256 / 360))

	if m.KeepRotation {
		dYaw = int32(m.Yaw)
		dPitch = int32(m.Pitch)
	}

	if dX < -128 || dX > 127 || dY < -128 || dY > 127 || dZ < -128 || dZ > 127 {
		tpkt := packets.TeleportEntityPacket{
			EntityId: e.GetEntityId(),
			X:        int32(math.Floor(m.X * 32)),
			Y:        int32(math.Floor(m.Y * 32)),
			Z:        int32(math.Floor(m.Z * 32)),
			Yaw:      byte(dYaw),
			Pitch:    byte(dPitch),
		}
		return tpkt.Serialize()
	}
	p := packets.EntityPositionAndRotationPacket{
		EntityId: e.GetEntityId(),
		X:        byte(dX),
		Y:        byte(dY),
		Z:        byte(dZ),
		Yaw:      byte(dYaw),
		Pitch:    byte(dPitch),
	}
	return p.Serialize()
}

func NewEntityVelocityPacket(entityId int32, m constants.MovementState) []byte {
	p := packets.EntityVelocityPacket{
		EntityId: entityId,
		Vx:       m.VelocityX,
		Vy:       m.VelocityY,
		Vz:       m.VelocityZ,
	}
	return p.Serialize()
}

func BroadcastContainerData(w *level.World, windowId byte, itemType, itemValue int16) {
	p := packets.ContainerDataPacket{
		WindowID: windowId,
		Type:     itemType,
		Value:    itemValue,
	}
	w.BroadcastPacket(p.Serialize())
}

func SendContainerData(connection net.Conn, windowId byte, itemType, itemValue int16) {
	p := packets.ContainerDataPacket{
		WindowID: windowId,
		Type:     itemType,
		Value:    itemValue,
	}
	connection.Write(p.Serialize())
}

func BroadcastSetSlot(w *level.World, windowId byte, slot int16, item inventory.Item) {
	p := packets.SetSlotPacket{
		WindowId: windowId,
		Slot:     slot,
		Item:     item,
	}
	w.BroadcastPacket(p.Serialize())
}

type SetTimePacket struct {
	Time int64
}

func (p *SetTimePacket) Serialize() []byte {
	w := packet.NewPacketWriter()
	w.WriteByte(packet.SetTime)
	w.WriteInt64(p.Time)
	return w.Bytes()
}

func BroadcastTime(w *level.World, tick int64) {
	p := SetTimePacket{Time: tick}
	w.BroadcastPacket(p.Serialize())
}

