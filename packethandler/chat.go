package packethandler

import (
	"log"
	"strings"

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
		return true
	}
	p.Message = "<" + pl.Username + "> " + message
	world.BroadcastPacket(p.Serialize())
	return false
}
