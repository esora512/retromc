package packethandler

import (
	"log"
	"net"

	"math"

	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

// sendSetSlot tells the client to update a single inventory slot.
func sendSetSlot(connection net.Conn, windowId byte, slot int16, item inventory.Item) {
	setSlotPacket := packets.SetSlotOutPacket{
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
			sendSetSlot(connection, 1, i, item)
		}
	}
}

func sendDispenserContents(connection net.Conn, dispenser *inventory.Dispenser) {
	for i := int16(0); i < int16(dispenser.Size); i++ {
		item := dispenser.PeekItem(i)
		if item.TypeId != -1 {
			sendSetSlot(connection, 1, i, item)
		}
	}
}

func sendFurnaceContents(connection net.Conn, furnace *inventory.Furnace) {
	for i := int16(0); i < int16(furnace.Size); i++ {
		item := furnace.PeekItem(i)
		if item.TypeId != -1 {
			sendSetSlot(connection, 1, i, item)
		}
	}
}

func broadcastChestContents(world *level.World, source *player.Player, chest *inventory.Chest) {
	world.ForEachPlayer(func(pl *player.Player) {
		if pl == source || pl.InventoryType != player.ChestInventory {
			return
		}
		if inventory.GetChest(pl.Chest.X, pl.Chest.Y, pl.Chest.Z) == chest {
			for i := int16(0); i < int16(chest.Size); i++ {
				sendSetSlot(pl.Connection, 1, i, chest.PeekItem(i))
			}
		}
	})
}

func broadcastDispenserContents(world *level.World, source *player.Player, dispenser *inventory.Dispenser) {
	world.ForEachPlayer(func(pl *player.Player) {
		if pl == source || pl.InventoryType != player.DispenserInventory {
			return
		}
		if inventory.GetDispenser(pl.Dispenser.X, pl.Dispenser.Y, pl.Dispenser.Z) == dispenser {
			for i := int16(0); i < int16(dispenser.Size); i++ {
				sendSetSlot(pl.Connection, 1, i, dispenser.PeekItem(i))
			}
		}
	})
}

func broadcastFurnaceContents(world *level.World, source *player.Player, furnace *inventory.Furnace) {
	world.ForEachPlayer(func(pl *player.Player) {
		if pl == source || pl.InventoryType != player.FurnaceInventory {
			return
		}
		if inventory.GetFurnace(pl.Furnace.X, pl.Furnace.Y, pl.Furnace.Z) == furnace {
			for i := int16(0); i < int16(furnace.Size); i++ {
				sendSetSlot(pl.Connection, 1, i, furnace.PeekItem(i))
			}
		}
	})
}

// presetInventory writes the starting items directly into the player's in-memory
// inventory. The caller is responsible for sending the inventory to the client.
func presetInventory(inv *inventory.Inventory) {
	return
	// inv.SetItem(36, constants.Rail.Value, 16, 0)
	// inv.SetItem(37, constants.Minecart.Value, 1, 0)
	// inv.SetItem(38, constants.Stone.Value, 64, 0)
	// inv.SetItem(39, constants.DiamondPickaxe.Value, 1, 0)
}

// sendChunks sends a 2x2 grid of chunks around the spawn point.
// Each chunk needs a PreChunk (init) followed by its MapChunk (data).
// Chunks are fetched from the world so any already-mutated state is preserved.
// func sendChunks(connection net.Conn, world *level.World) {
// 	for cx := int32(-1); cx <= 0; cx++ {
// 		for cz := int32(-1); cz <= 0; cz++ {
// 			// pre-chunk: uses chunk coordinates
// 			preChunkPacket := packets.PreChunkOutPacket{
// 				X:    cx,
// 				Z:    cz,
// 				Mode: true,
// 			}
// 			connection.Write(preChunkPacket.Serialize())

// 			// map-chunk: X/Z are block coordinates of the chunk's origin
// 			chunk := world.GetOrCreateChunk(cx, cz, level.Template)

// 			mapChunkPacket := packets.MapChunkOutPacket{}
// 			mapChunkPacket.Apply(*chunk)
// 			connection.Write(mapChunkPacket.Serialize())
// 		}
// 	}
// }

var WORLD_RANGE = 8

func GenerateSquareWorld(world *level.World) {
	for cx := -WORLD_RANGE; cx <= WORLD_RANGE; cx++ {
		for cz := -WORLD_RANGE; cz <= WORLD_RANGE; cz++ {
			world.GetOrCreateChunk(int32(cx), int32(cz), level.Template)
		}
	}
}

func SendLoadedChunks(conn net.Conn, world *level.World, pl *player.Player) {
	playerChunkX := int32(math.Floor(float64(pl.X) / 16))
	playerChunkZ := int32(math.Floor(float64(pl.Z) / 16))
	const radius = 2
	for coord, chunk := range world.LoadChunks() {
		if chunk == nil {
			continue
		}
		dx := coord.X - playerChunkX
		dz := coord.Z - playerChunkZ
		if dx < -radius || dx > radius || dz < -radius || dz > radius {
			continue
		}
		pre := packets.PreChunkOutPacket{X: coord.X, Z: coord.Z, Mode: true}
		conn.Write(pre.Serialize())
		mapChunk := packets.MapChunkOutPacket{}
		mapChunk.Apply(*chunk)
		conn.Write(mapChunk.Serialize())
	}
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

func sendInventory(connection net.Conn, pl *player.Player) {
	loaded, err := player.LoadInventory(pl.Username, &pl.Inventory)
	if err != nil {
		log.Printf("Failed to load inventory for %s: %v", pl.Username, err)
	}
	if !loaded {
		presetInventory(&pl.Inventory)
	}
	windowItemsPacket := packets.WindowItemsOutPacket{
		WindowId: 0, // 0 = player inventory
		Count:    int16(pl.Inventory.Size),
		Payload:  pl.Inventory,
	}
	connection.Write(windowItemsPacket.Serialize())
}

func sendPlayerPositionAndLook(connection net.Conn) {
	const spawnY = 64.0
	packet := packets.PlayerPositionAndLookOutPacket{
		X:        0,
		Y:        spawnY,
		Stance:   spawnY + 2, // Stance MUST be Y + eye height; if Stance < Y client looks up
		Z:        0,
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
	tpkt := packets.TeleportEntity{
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
	tpkt := packets.TeleportEntity{
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
			selfPkt := packets.PlayerPositionAndLookOutPacket{
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

	p := packets.EntityPositionOutPacket{
		EntityId: c.GetEntityId(),
		X:        byte(dX),
		Y:        byte(dY),
		Z:        byte(dZ),
	}
	w.BroadcastPacket(p.Serialize())
}

func BroadcastRelativePosition(w *level.World, c level.Entity, prevX, prevY, prevZ, nextX, nextY, nextZ float64, yaw byte) {
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

	p := packets.EntityPositionAndLookOutPacket{
		EntityId: c.GetEntityId(),
		X:        byte(dX),
		Y:        byte(dY),
		Z:        byte(dZ),
		Yaw:      yaw,
		Pitch:    0,
	}
	w.BroadcastPacket(p.Serialize())
}
