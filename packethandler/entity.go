package packethandler

import (
	"log"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func handleInteractWithEntityInPacket(p packets.InteractWithEntityOutPacket, pl *player.Player, world *level.World) {
	player := world.Players[p.PlayerId]
	other := world.Players[p.EntityId]
	log.Printf("%s interacted with %s", player.Username, other.Username)
}
