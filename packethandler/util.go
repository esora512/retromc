package packethandler

import (
	"net"

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
	// inv.SetItem(36, constants.IronPickaxe.Value, 1, 0)
	// inv.SetItem(37, constants.Stone.Value, 64, 0)
	// inv.SetItem(38, constants.Planks.Value, 64, 0)
	// inv.SetItem(40, constants.String.Value, 32, 0)
	// inv.SetItem(41, constants.Dispenser.Value, 16, 0)
	// inv.SetItem(38, 326, 1) // Water Bucket
	// inv.SetItem(39, 327, 1) // Lava Bucket
}

// sendChunks sends a 2x2 grid of chunks around the spawn point.
// Each chunk needs a PreChunk (init) followed by its MapChunk (data).
// Chunks are fetched from the world so any already-mutated state is preserved.
func sendChunks(connection net.Conn, world *level.World) {
	for cx := int32(-1); cx <= 0; cx++ {
		for cz := int32(-1); cz <= 0; cz++ {
			// pre-chunk: uses chunk coordinates
			preChunkPacket := packets.PreChunkOutPacket{
				X:    cx,
				Z:    cz,
				Mode: true,
			}
			connection.Write(preChunkPacket.Serialize())

			// map-chunk: X/Z are block coordinates of the chunk's origin
			chunk := world.GetOrCreateChunk(cx, cz)

			mapChunkPacket := packets.MapChunkOutPacket{}
			mapChunkPacket.Apply(*chunk)
			connection.Write(mapChunkPacket.Serialize())
		}
	}
}

func sendSpawnPosition(connection net.Conn) {
	spawnPositionPacket := packets.SpawnPositionOutPacket{
		X: 0,
		Y: 64,
		Z: 0,
	}
	outData := spawnPositionPacket.Serialize()
	connection.Write(outData)
}

func sendInventory(connection net.Conn, pl *player.Player) {
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
