package packethandler

import (
	"log"
	"strings"

	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func handleChatMessageInPacket(p packets.ChatMessagePacket, pl *player.Player, world *level.World) bool {
	message := p.Message
	if strings.HasPrefix(message, "/") {
		log.Printf("Command: %s", message)
		if strings.HasPrefix(message, "/give") {
			command := strings.TrimPrefix(message, "/give ")
			before := pl.Inventory.PeekItem(pl.HotbarSlot)
			pl.GivePlayer(command)
			after := pl.Inventory.PeekItem(pl.HotbarSlot)
			if before != after {
				sendEquipmentChangeForHotbarSlot(world, pl)
			}
		}
		if strings.HasPrefix(message, "/save") {
			if err := world.SaveChanges("world.dat"); err != nil {
				log.Println("Failed to save world:", err)
			} else {
				log.Println("World saved successfully.")
			}
			if err := player.SaveInventory(pl.Username, pl.Inventory); err != nil {
				log.Println("Failed to save player data:", err)
			} else {
				log.Println("Player data saved successfully.")
			}
			if err := inventory.SaveContainers("containers.dat"); err != nil {
				log.Println("Failed to save containers:", err)
			} else {
				log.Println("Containers saved successfully.")
			}
		}

		return true
	}
	p.Message = "<" + pl.Username + "> " + message
	world.BroadcastPacket(p.Serialize())
	return false
}
