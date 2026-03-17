package packethandler

import (
	"net"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/packet/packets"
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

// presetInventory writes the starting items directly into the player's in-memory
// inventory. The caller is responsible for sending the inventory to the client.
func presetInventory(inv *inventory.Inventory) {
	inv.SetItem(36, constants.IronPickaxe.Value, 1, 0) // Iron Pickaxe
	inv.SetItem(37, constants.Stone.Value, 64, 0)      // Stone x64
	inv.SetItem(38, constants.Planks.Value, 64, 0)     // Planks x64
	inv.SetItem(39, constants.Leather.Value, 64, 0)
	//inv.SetItem(38, 326, 1) // Water Bucket
	//inv.SetItem(39, 327, 1) // Lava Bucket
}
