package packethandler

import (
	"log"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func handleInteractWithEntityInPacket(p packets.InteractWithEntityOutPacket, pl *player.Player, world *level.World) {
	player := world.Players[p.PlayerId]
	other := world.Entities[p.EntityId]
	log.Printf("%s interacted with %s", player.Username, other.GetName())

	if p.Attack {
		oldHP := other.GetHP()
		newHP := oldHP - 1
		other.SetHP(newHP)
		log.Printf("%s attacked %s for 1 damage (HP: %d -> %d)", player.Username, other.GetName(), oldHP, newHP)
		if newHP <= 0 {
			log.Printf("%s killed %s", player.Username, other.GetName())
			if other.IsRideable() {
				ridable, _ := other.(*entities.RideableEntity)
				if ridable.ObjectType == constants.ObjectBoat {
					slot := pl.Inventory.AddItem(constants.Boat.Value, 0, 1)
					if slot < 0 {
						return
					}
					sendSetSlot(pl.Connection, 0, slot, pl.Inventory.Items[slot])
					
				}
				if ridable.ObjectType == constants.ObjectMinecart {
					slot := pl.Inventory.AddItem(constants.Minecart.Value, 0, 1)
					if slot < 0 {
						return
					}
					sendSetSlot(pl.Connection, 0, slot, pl.Inventory.Items[slot])
				}

			}
		}
		return
	}

	world.MulticastPacket(packets.ArmSwing(pl), pl)
	if other.IsRideable() {
		ridable, _ := other.(*entities.RideableEntity)
		if pl.IsRiding != -1 {
			pl.IsRiding = -1
			world.BroadcastPacket(packets.PlayerEntityMetadataPacketRiding(pl, false))
			world.BroadcastPacket(packets.AlicesRidesBob(pl.GetEntityId(), -1))
			ridable.PassengerEntityId = -1
		} else {
			world.BroadcastPacket(packets.PlayerEntityMetadataPacketRiding(pl, true))
			world.BroadcastPacket(packets.AlicesRidesBob(pl.GetEntityId(), other.GetEntityId()))
			pl.IsRiding = other.GetEntityId()
			ridable.PassengerEntityId = pl.GetEntityId()
			pl.Lx = pl.X
			pl.Ly = pl.Y
			pl.Lz = pl.Z
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
