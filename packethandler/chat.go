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
			pl.GivePlayer(command)
		}
		return true
	}
	p.Message = "<" + pl.Username + "> " + message
	world.BroadcastPacket(p.Serialize())
	return false
}
