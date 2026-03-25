package packethandler

import (
	"net"

	"github.com/leNicDev/retromc/constants"
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
	inv.SetItem(36, constants.IronPickaxe.Value, 1, 0)
	inv.SetItem(37, constants.Stone.Value, 64, 0)
	inv.SetItem(38, constants.Planks.Value, 64, 0)
	inv.SetItem(40, constants.String.Value, 32, 0)
	inv.SetItem(41, constants.Dispenser.Value, 16, 0)
	//inv.SetItem(38, 326, 1) // Water Bucket
	//inv.SetItem(39, 327, 1) // Lava Bucket
}
