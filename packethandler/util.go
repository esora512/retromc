package packethandler

import (
	"fmt"
	"net"

	"math"

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
		if world.GetChest(pl.Chest.X, pl.Chest.Y, pl.Chest.Z) == chest {
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
		if world.GetDispenser(pl.Dispenser.X, pl.Dispenser.Y, pl.Dispenser.Z) == dispenser {
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
		if world.GetFurnace(pl.Furnace.X, pl.Furnace.Y, pl.Furnace.Z) == furnace {
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

const VIEW_DISTANCE = 15

func initialUpdateChunks(world *level.World, x, z float64, pl *player.Player) {
	const VIEW_DISTANCE = 3
	cx := level.WorldToChunkCoord(int32(x))
	cz := level.WorldToChunkCoord(int32(z))

	// Nothing to do if we haven't crossed a chunk boundary
	if cx == pl.LastChunkX && cz == pl.LastChunkZ && pl.HasInitializedChunks {
		return
	}

	wanted := make(map[string]level.ChunkCoord)
	for dx := -VIEW_DISTANCE; dx <= VIEW_DISTANCE; dx++ {
		for dz := -VIEW_DISTANCE; dz <= VIEW_DISTANCE; dz++ {
			coord := level.ChunkCoord{X: cx + int32(dx), Z: cz + int32(dz)}
			wanted[coord.String()] = coord
		}
	}

	for _, off := range spiralOffsets(VIEW_DISTANCE) {
		coord := level.ChunkCoord{X: cx + off.X, Z: cz + off.Z}
		key := coord.String()

		if pl.SentChunks.Has(key) {
			continue
		}

		chunk := world.GetOrCreateChunk(coord.X, coord.Z, pl.Dimension)

		pre := packets.SetChunkVisibilityPacket{X: coord.X, Z: coord.Z, Mode: true}
		pl.Connection.Write(pre.Serialize())

		mapChunk := packets.ChunkBlockRegionPacket{}
		mapChunk.Apply(*chunk)
		pl.Connection.Write(mapChunk.Serialize())

		pl.SentChunks.Set(key, coord.X, coord.Z)
	}

	// Unload chunks that fell out of range
	for key, coord := range pl.SentChunks {
		if _, ok := wanted[key]; !ok {
			unload := packets.SetChunkVisibilityPacket{X: coord.X, Z: coord.Z, Mode: false}
			pl.Connection.Write(unload.Serialize())
			delete(pl.SentChunks, key)
		}
	}
	pl.LastChunkX = cx
	pl.LastChunkZ = cz
	pl.HasInitializedChunks = true
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

func updateChunks(world *level.World, x, z float64, pl *player.Player) {
	cx := level.WorldToChunkCoord(int32(x))
	cz := level.WorldToChunkCoord(int32(z))

	// Nothing to do if we haven't crossed a chunk boundary
	if cx == pl.LastChunkX && cz == pl.LastChunkZ && pl.HasInitializedChunks {
		return
	}

	wanted := make(map[string]level.ChunkCoord)
	for dx := -VIEW_DISTANCE; dx <= VIEW_DISTANCE; dx++ {
		for dz := -VIEW_DISTANCE; dz <= VIEW_DISTANCE; dz++ {
			coord := level.ChunkCoord{X: cx + int32(dx), Z: cz + int32(dz)}
			wanted[coord.String()] = coord
		}
	}

	for _, off := range spiralOffsets(VIEW_DISTANCE) {
		coord := level.ChunkCoord{X: cx + off.X, Z: cz + off.Z}
		key := coord.String()

		if pl.SentChunks.Has(key) {
			continue
		}

		chunk := world.GetOrCreateChunk(coord.X, coord.Z, pl.Dimension)

		pre := packets.SetChunkVisibilityPacket{X: coord.X, Z: coord.Z, Mode: true}
		pl.Connection.Write(pre.Serialize())

		mapChunk := packets.ChunkBlockRegionPacket{}
		mapChunk.Apply(*chunk)
		pl.Connection.Write(mapChunk.Serialize())

		pl.SentChunks.Set(key, coord.X, coord.Z)
	}

	// Unload chunks that fell out of range
	for key, coord := range pl.SentChunks {
		if _, ok := wanted[key]; !ok {
			unload := packets.SetChunkVisibilityPacket{X: coord.X, Z: coord.Z, Mode: false}
			pl.Connection.Write(unload.Serialize())
			delete(pl.SentChunks, key)
		}
	}
	pl.LastChunkX = cx
	pl.LastChunkZ = cz
	pl.HasInitializedChunks = true
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
		Stance:   y + 2, // Stance MUST be Y + eye height; if Stance < Y client looks up
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

// Use teleport packet to obtain absolute control over minecart
// Too bad at math to get it to work with relative positions and mimicking client-side calculations...
func BroadcastTeleport(w *level.World, c level.Entity, cx, cy, cz float64, yaw byte) {
	tpkt := packets.TeleportEntityPacket{
		EntityId: c.GetEntityId(),
		X:        int32(math.Floor(cx * 32)),
		Y:        int32(math.Floor(cy * 32)),
		Z:        int32(math.Floor(cz * 32)),
		Yaw:      yaw,
		Pitch:    0,
	}
	w.BroadcastPacket(tpkt.Serialize())
}

func BroadcastTeleportPlayer(w *level.World, c level.Entity, cx, cy, cz float64, yaw byte) {
	tpkt := packets.TeleportEntityPacket{
		EntityId: c.GetEntityId(),
		X:        int32(math.Floor(cx * 32)),
		Y:        int32(math.Floor(cy * 32)),
		Z:        int32(math.Floor(cz * 32)),
		Yaw:      yaw,
		Pitch:    0,
	}
	data := tpkt.Serialize()

	w.Mu.RLock()
	defer w.Mu.RUnlock()
	for _, pl := range w.Players {
		if !pl.LoggedIn {
			continue
		}
		if pl.GetEntityId() == c.GetEntityId() {
			selfPkt := packets.PlayerPositionAndRotationPacket{
				X: cx, Y: cy, Z: cz, Stance: cy + 2, OnGround: true,
				Yaw:   float32(yaw) * 360.0 / 256.0,
				Pitch: 0,
			}
			pl.Connection.Write(selfPkt.Serialize())
			continue
		}
		pl.Connection.Write(data)
	}
}

func BroadcastPosition(w *level.World, c level.Entity, prevX, prevY, prevZ, nextX, nextY, nextZ float64) {
	encPrevX := int32(math.Floor(prevX * 32))
	encPrevY := int32(math.Floor(prevY * 32))
	encPrevZ := int32(math.Floor(prevZ * 32))
	encNextX := int32(math.Floor(nextX * 32))
	encNextY := int32(math.Floor(nextY * 32))
	encNextZ := int32(math.Floor(nextZ * 32))

	dX := encNextX - encPrevX
	dY := encNextY - encPrevY
	dZ := encNextZ - encPrevZ

	// delta overflow guard — fall back to teleport if moved more than 4 blocks
	if dX < -128 || dX > 127 || dY < -128 || dY > 127 || dZ < -128 || dZ > 127 {
		BroadcastTeleport(w, c, nextX, nextY, nextZ, 0)
		return
	}

	p := packets.EntityPositionPacket{
		EntityId: c.GetEntityId(),
		X:        byte(dX),
		Y:        byte(dY),
		Z:        byte(dZ),
	}
	w.BroadcastPacket(p.Serialize())
}

func NewTeleportPacket(w *level.World, e level.Entity, m constants.MovementState) []byte {
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

func NewPositionOrTeleportPacket(w *level.World, e level.Entity, m constants.MovementState) []byte {
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

func NewRotationPacket(w *level.World, e level.Entity, m constants.MovementState) []byte {
	dYaw := int32(math.Floor(float64(m.Yaw) * 256 / 360))
	dPitch := int32(math.Floor(float64(m.Pitch) * 256 / 360))

	p := packets.EntityRotationPacket{
		EntityId: e.GetEntityId(),
		Yaw:      byte(dYaw),
		Pitch:    byte(dPitch),
	}
	return p.Serialize()
}

func NewPositionAndRotationOrTeleportPacket(w *level.World, e level.Entity, m constants.MovementState) []byte {
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

func BroadcastPositionAndRotation(w *level.World, c level.Entity, prevX, prevY, prevZ, nextX, nextY, nextZ float64, yaw byte) {
	encPrevX := int32(math.Floor(prevX * 32))
	encPrevY := int32(math.Floor(prevY * 32))
	encPrevZ := int32(math.Floor(prevZ * 32))
	encNextX := int32(math.Floor(nextX * 32))
	encNextY := int32(math.Floor(nextY * 32))
	encNextZ := int32(math.Floor(nextZ * 32))

	dX := encNextX - encPrevX
	dY := encNextY - encPrevY
	dZ := encNextZ - encPrevZ

	// delta overflow guard — fall back to teleport if moved more than 4 blocks
	if dX < -128 || dX > 127 || dY < -128 || dY > 127 || dZ < -128 || dZ > 127 {
		BroadcastTeleport(w, c, nextX, nextY, nextZ, 0)
		return
	}

	p := packets.EntityPositionAndRotationPacket{
		EntityId: c.GetEntityId(),
		X:        byte(dX),
		Y:        byte(dY),
		Z:        byte(dZ),
		Yaw:      yaw,
		Pitch:    0,
	}
	w.BroadcastPacket(p.Serialize())
}

func BroadcastEntityVelocity(w *level.World, entityId int32, vx, vy, vz float64) {
	packet := packets.EntityVelocityPacket{
		EntityId: entityId,
		Vx:       vx,
		Vy:       vy,
		Vz:       vz,
	}
	w.BroadcastPacket(packet.Serialize())
}

func NewEntityVelocityPacket(entityId int32, vx, vy, vz float64) []byte {
	p := packets.EntityVelocityPacket{
		EntityId: entityId,
		Vx:       vx,
		Vy:       vy,
		Vz:       vz,
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

func BroadcastWakeUp(w *level.World, id int32) {
	p := packets.AnimationPacket{PlayerId: id, Animation: 3}
	w.BroadcastPacket(p.Serialize())
}

type SpawnObject struct {
	EntityId      int32
	ObjectType    byte
	X             int32
	Y             int32
	Z             int32
	OwnerEntityId int32
	VelocityX     int16
	VelocityY     int16
	VelocityZ     int16
}

func (p *SpawnObject) Serialize() []byte {
	writer := packet.NewPacketWriter()
	writer.WriteByte(packet.SpawnObject)
	writer.WriteInt32(p.EntityId)
	writer.WriteByte(p.ObjectType)
	writer.WriteInt32(p.X)
	writer.WriteInt32(p.Y)
	writer.WriteInt32(p.Z)
	writer.WriteInt32(p.OwnerEntityId)
	writer.WriteInt16(p.VelocityX)
	writer.WriteInt16(p.VelocityY)
	writer.WriteInt16(p.VelocityZ)
	return writer.Bytes()
}

func BroadcastSpawnObject(w *level.World, eId int32, oType byte, x, y, z, oeId int32, velX, velY, velZ int16) {
	p := SpawnObject{
		EntityId:      eId,
		ObjectType:    oType,
		X:             x,
		Y:             y,
		Z:             z,
		OwnerEntityId: oeId,
		VelocityX:     velX,
		VelocityY:     velY,
		VelocityZ:     velZ,
	}
	w.BroadcastPacket(p.Serialize())
}
