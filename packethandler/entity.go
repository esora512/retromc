package packethandler

import (
	"log"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func dmgGiven(typeId int16) int16 {
	if typeId == constants.WoodenAxe.Value || typeId == constants.GoldAxe.Value {
		return 3
	}
	if typeId == constants.WoodenShovel.Value || typeId == constants.GoldShovel.Value {
		return 1
	}
	if typeId == constants.WoodenSword.Value || typeId == constants.GoldSword.Value {
		return 4
	}
	if typeId == constants.WoodenPickaxe.Value || typeId == constants.GoldPickaxe.Value {
		return 2
	}

	if typeId == constants.StoneSword.Value {
		return 6
	}

	if typeId == constants.IronSword.Value {
		return 8
	}

	if typeId == constants.DiamondSword.Value {
		return 10
	}

	if typeId == constants.StoneAxe.Value {
		return 5
	}

	if typeId == constants.IronAxe.Value {
		return 8
	}

	if typeId == constants.DiamondAxe.Value {
		return 9
	}

	if typeId == constants.StonePickaxe.Value {
		return 4
	}

	if typeId == constants.IronPickaxe.Value {
		return 6
	}

	if typeId == constants.DiamondPickaxe.Value {
		return 8
	}

	if typeId == constants.StoneShovel.Value {
		return 3
	}

	if typeId == constants.IronShovel.Value {
		return 5
	}

	if typeId == constants.DiamondShovel.Value {
		return 7
	}

	return 1
}

func dmgReduced(items []inventory.Item, dmg int16) int16 {
	newDmg := dmg
	if items[5].TypeId != -1 {
		newDmg -= 3
	}
	if items[6].TypeId != -1 {
		newDmg -= 8
	}
	if items[7].TypeId != -1 {
		newDmg -= 6
	}
	if items[8].TypeId != -1 {
		newDmg -= 3
	}
	return newDmg
}

func handleInteractWithEntityInPacket(p packets.InteractWithEntityOutPacket, pl *player.Player, world *level.World) {
	player := world.Players[p.PlayerId]
	other := world.Entities[p.EntityId]
	log.Printf("%s interacted with %s", player.Username, other.GetName())

	if p.Attack {
		oldHP := other.GetHP()
		item := pl.Inventory.Items[pl.HotbarSlot]
		log.Printf("%s has %d in hand", pl.Username, item.TypeId)
		dmg := int16(1)
		if item.TypeId != -1 {
			dmg = dmgGiven(item.TypeId)
		}
		if other.IsPlayer() {
			otherPlayer := world.Players[other.GetEntityId()]
			dmg = dmgReduced(otherPlayer.Inventory.Items, dmg)
			sendSetHealth(otherPlayer.Connection, uint16(oldHP-dmg))
			p := packets.EntityEventOutPacket{
				EntityId: other.GetEntityId(),
				Action:   2,
			}
			world.BroadcastPacket(p.Serialize())
		}
		newHP := oldHP - dmg
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
