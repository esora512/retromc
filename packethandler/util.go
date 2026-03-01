package packethandler

import (
	"net"

	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func setItemInInventory(connection net.Conn, typeId int16, count int, slot int16) {
	setSlotPacket := packets.SetSlotOutPacket{
		WindowId: 0,    // 0 = player inventory
		Slot:     slot, // slots 36-44 are the hotbar; 36 is the first
		Item:     player.NewItem(typeId, byte(count)),
	}
	outData := setSlotPacket.Serialize()
	connection.Write(outData)
}

func presetInventory(connection net.Conn) {
	setItemInInventory(connection, 257, 1, 36) // Pickaxe
	setItemInInventory(connection, 1, 64, 37)  // Stones
	setItemInInventory(connection, 326, 1, 38) // Water Bucket
	setItemInInventory(connection, 327, 1, 39) // Lava Bucket
}
