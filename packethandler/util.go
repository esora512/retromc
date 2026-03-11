package packethandler

import (
	"net"

	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

// sendSetSlot tells the client to update a single inventory slot.
func sendSetSlot(connection net.Conn, windowId byte, slot int16, item player.Item) {
	setSlotPacket := packets.SetSlotOutPacket{
		WindowId: windowId,
		Slot:     slot,
		Item:     item,
	}
	connection.Write(setSlotPacket.Serialize())
}

// presetInventory writes the starting items directly into the player's in-memory
// inventory. The caller is responsible for sending the inventory to the client.
func presetInventory(inv *player.Inventory) {
	inv.SetItem(36, 257, 1) // Iron Pickaxe
	inv.SetItem(37, 1, 64)  // Stone x64
	inv.SetItem(38, 5, 64)  // Planks x64

	//inv.SetItem(38, 326, 1) // Water Bucket
	//inv.SetItem(39, 327, 1) // Lava Bucket
}
