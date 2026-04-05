package packethandler

import (
	"log"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func handleInteractWithEntityInPacket(p packets.InteractWithEntityOutPacket, pl *player.Player, world *level.World) {
	player := world.Players[p.PlayerId]
	other := world.Entities[p.EntityId]
	log.Printf("%s interacted with %s", player.Username, other.GetName())

	world.MulticastPacket(packets.ArmSwing(pl), pl)
	if other.IsRideable() {
		if pl.IsRiding {
			pl.Connection.Write(packets.PlayerEntityMetadataPacketRiding(pl, false))
			world.BroadcastPacket(packets.AlicesRidesBob(pl.GetEntityId(), -1))
			pl.IsRiding = false
		} else {
			pl.Connection.Write(packets.PlayerEntityMetadataPacketRiding(pl, true))
			world.BroadcastPacket(packets.AlicesRidesBob(pl.GetEntityId(), other.GetEntityId()))
			pl.IsRiding = true
		}
	}
}

func handleEntityActionInPacket(p packets.EntityActionInPacket, pl *player.Player, world *level.World) {
	if p.ActionId == 1 {
		world.MulticastPacket(packets.PlayerEntityMetadataPacketSneak(pl, true), pl)
	}
	if p.ActionId == 2 {
		world.MulticastPacket(packets.PlayerEntityMetadataPacketSneak(pl, false), pl)
	}
}
