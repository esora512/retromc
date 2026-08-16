package packethandler

import (
	"log"
	"math"
	"math/rand"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/crafting"
	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func DropInventory(
	world *level.World,
	inv *inventory.Inventory,
	x, y, z float64,
	dim int32,
) {
	for i := range inv.Items {
		stack := &inv.Items[i]

		if stack.TypeId == -1 {
			continue
		}

		// Same as:
		// rand.NextFloat() * 0.8f + 0.1f
		offsetX := rand.Float64()*0.8 + 0.1
		offsetY := rand.Float64()*0.8 + 0.1
		offsetZ := rand.Float64()*0.8 + 0.1

		remaining := int(stack.Count)

		for remaining > 0 {
			countDecrement := rand.Intn(21) + 10

			if countDecrement > remaining {
				countDecrement = remaining
			}

			remaining -= countDecrement

			spawnX := x + offsetX
			spawnY := y + offsetY
			spawnZ := z + offsetZ

			velocity := 0.05

			velX := rand.Float64() * velocity
			velY := rand.Float64()*velocity + 0.2
			velZ := rand.Float64() * velocity

			CreateDroppedItem(
				world,
				spawnX,
				spawnY,
				spawnZ,
				int32(stack.TypeId),
				byte(countDecrement),
				byte(stack.Metadata),
				velX,
				velY,
				velZ,
				60,
				dim,
			)
		}
		inv.Items[i] = inventory.EmptyItem()
	}
}

func quantizeSpawnVelocity(v float64) int8 {
	return int8(v * 128.0)
}

func CreateDroppedItem(w *level.World, x, y, z float64, itemId int32, amount, meta byte, velX, velY, velZ float64, pickupDelay, dim int32) int32 {
	entityId := w.AddDroppedItem(x, y, z, itemId, amount, meta, pickupDelay, dim)

	if item, ok := w.Entities[entityId].(*level.DroppedItem); ok {
		item.VelX = velX
		item.VelY = velY
		item.VelZ = velZ
	}

	spawn := packets.SpawnItemPacket{
		EntityId: entityId,
		ItemId:   int16(itemId),
		Amount:   amount,
		Metadata: meta,
		X:        int32(math.Floor(x * 32)),
		Y:        int32(math.Floor(y * 32)),
		Z:        int32(math.Floor(z * 32)),
		Pitch:    byte(quantizeSpawnVelocity(velX)),
		Yaw:      byte(quantizeSpawnVelocity(velY)),
		Roll:     byte(quantizeSpawnVelocity(velZ)),
	}
	w.BroadcastPacket(spawn.Serialize())
	velocityPacket := packets.EntityVelocityPacket{EntityId: entityId, Vx: velX, Vy: velY, Vz: velZ}
	w.BroadcastPacket(velocityPacket.Serialize())
	return entityId
}

func dmgGiven(typeId int16) (int16, bool) {
	if typeId == constants.WoodenAxe.Value || typeId == constants.GoldAxe.Value {
		return 3, true
	}
	if typeId == constants.WoodenShovel.Value || typeId == constants.GoldShovel.Value {
		return 1, true
	}
	if typeId == constants.WoodenSword.Value || typeId == constants.GoldSword.Value {
		return 4, true
	}
	if typeId == constants.WoodenPickaxe.Value || typeId == constants.GoldPickaxe.Value {
		return 2, true
	}

	if typeId == constants.StoneSword.Value {
		return 6, true
	}

	if typeId == constants.IronSword.Value {
		return 8, true
	}

	if typeId == constants.DiamondSword.Value {
		return 10, true
	}

	if typeId == constants.StoneAxe.Value {
		return 5, true
	}

	if typeId == constants.IronAxe.Value {
		return 8, true
	}

	if typeId == constants.DiamondAxe.Value {
		return 9, true
	}

	if typeId == constants.StonePickaxe.Value {
		return 4, true
	}

	if typeId == constants.IronPickaxe.Value {
		return 6, true
	}

	if typeId == constants.DiamondPickaxe.Value {
		return 8, true
	}

	if typeId == constants.StoneShovel.Value {
		return 3, true
	}

	if typeId == constants.IronShovel.Value {
		return 5, true
	}

	if typeId == constants.DiamondShovel.Value {
		return 7, true
	}

	return 1, false
}

func dmgReduced(world *level.World, pl *player.Player, items []inventory.Item, dmg int16) int16 {
	checkArmor := func(slot int, dmg, reduction int16) int16 {
		if items[slot].TypeId != -1 {
			newDmg := dmg
			items[slot].Metadata++
			SendSetSlot(pl.Connection, 0, int16(slot), items[slot])
			if crafting.Durability(items[slot].TypeId) <= items[slot].Metadata {
				items[slot] = inventory.EmptyItem()
				SendSetSlot(pl.Connection, 0, int16(slot), inventory.EmptyItem())
				sendSetEquipment(world, int16(slot), -1, pl.GetEntityId())
			}
			if reduction > newDmg {
				return 0
			}
			newDmg -= reduction
			return newDmg
		}
		return dmg
	}

	newDmg := checkArmor(5, dmg, 3)   // Helmet
	newDmg = checkArmor(6, newDmg, 8) // Chestplate
	newDmg = checkArmor(7, newDmg, 6) // Leggings
	newDmg = checkArmor(8, newDmg, 3) // Boots
	return newDmg
}

const (
	knockbackVelocityDampening = 0.5
	knockbackHorizontal        = 0.4
	knockbackVertical          = 0.4
)

func applyKnockback(w *level.World, attacker, victim level.Entity) {
	aX, _, aZ := attacker.GetPosition()
	viX, _, viZ := victim.GetPosition()
	dx := aX - viX
	dz := aZ - viZ
	dist := math.Sqrt(dx*dx + dz*dz)

	if dist < 1e-4 {
		dx = (rand.Float64() - rand.Float64()) * 0.01
		dz = (rand.Float64() - rand.Float64()) * 0.01
		dist = math.Sqrt(dx*dx + dz*dz)
	}
	dx /= dist
	dz /= dist

	vX, vY, vZ := victim.GetVelocity()
	vX *= knockbackVelocityDampening
	vZ *= knockbackVelocityDampening
	vX -= dx * knockbackHorizontal
	vZ -= dz * knockbackHorizontal
	vY = math.Min(vY+knockbackVertical, knockbackVertical)

	ev := packets.EntityVelocityPacket{
		EntityId: victim.GetEntityId(),
		Vx:       vX,
		Vy:       vY,
		Vz:       vZ,
	}
	if mob, ok := victim.(*level.Mob); ok {
		mob.ApplyKnockback(vX, vY, vZ)
	}
	w.BroadcastPacket(ev.Serialize())
}

func handleInteractWithEntityPacket(p packets.InteractWithEntityPacket, pl *player.Player, world *level.World, tracker *level.EntityTracker) {
	var ok bool
	log.Printf("%d attacks %d", p.PlayerId, p.EntityId)
	player, ok := world.Players[p.PlayerId]
	other, ok := world.Entities[p.EntityId]
	if !ok {
		return
	}
	//log.Printf("%s interacted with %s", player.Username, other.GetName())

	if p.Attack {
		oldHP := other.GetHP()
		item := pl.Inventory.Items[pl.HotbarSlot]
		log.Printf("%s attacks %s", player.GetName(), other.GetName())
		dmg := int16(1)
		given := false
		if item.TypeId != -1 {
			dmg, given = dmgGiven(item.TypeId)
			if given {
				item.Metadata++
				SendSetSlot(pl.Connection, 0, pl.HotbarSlot, item)
				if crafting.Durability(item.TypeId) <= item.Metadata {
					item = inventory.EmptyItem()
					SendSetSlot(pl.Connection, 0, pl.HotbarSlot, item)
				}
				pl.Inventory.Items[pl.HotbarSlot] = item
			}
		}
		if other.IsPlayer() {
			otherPlayer := world.Players[other.GetEntityId()]
			dmg = dmgReduced(world, otherPlayer, otherPlayer.Inventory.Items, dmg)
			SendSetHealth(otherPlayer.Connection, uint16(oldHP-dmg))
			BroadcastPain(world, other.GetEntityId())
		} else if other.IsMob() {
			BroadcastPain(world, other.GetEntityId())
			if mob, ok := other.(*level.Mob); ok {
				mob.SetTargetForced(pl.GetEntityId())
			}
		}

		newHP := oldHP - dmg
		other.SetHP(newHP)
		//log.Printf("%s attacked %s for 1 damage (HP: %d -> %d)", player.Username, other.GetName(), oldHP, newHP)
		if other.IsPlayer() || other.IsMob() {
			applyKnockback(world, pl, other)

		}

		if other.IsRideable() {
			p := packets.EntityEventPacket{
				EntityId: other.GetEntityId(),
				Action:   2,
			}
			world.BroadcastPacket(p.Serialize())
		}

		if newHP <= 0 {
			if other.IsPlayer() {
				cMsgPkt := packets.ChatMessagePacket{
					Message: other.GetName() + " was killed by " + player.Username,
				}
				world.BroadcastPacket(cMsgPkt.Serialize())

				x, y, z := other.GetPosition()
				otherPl, _ := world.Players[other.GetEntityId()]
				DropInventory(world, &otherPl.Inventory, x, y, z, otherPl.GetDim())

				tracker.ResetViewer(other.GetEntityId())
			}
			p := packets.EntityEventPacket{
				EntityId: other.GetEntityId(),
				Action:   3,
			}
			world.BroadcastPacket(p.Serialize())

			if other.IsRideable() {
				ridable, _ := other.(*entities.RideableEntity)
				if ridable.ObjectType == constants.ObjectBoat {
					x, y, z := other.GetPosition()
					world.BroadcastDroppedItem(x, y, z, constants.Boat.Value, 0, 1, other.GetDim(), 5, tracker)

				}
				if ridable.ObjectType == constants.ObjectMinecart {
					x, y, z := other.GetPosition()
					world.BroadcastDroppedItem(x, y, z, constants.Minecart.Value, 0, 1, other.GetDim(), 5, tracker)

				}
			}

			if other.IsMob() {
				m, _ := other.(*level.Mob)
				if m.MobType == 52 {
					x, y, z := other.GetPosition()
					world.BroadcastDroppedItem(x, y, z, constants.String.Value, 0, 1, other.GetDim(), 5, tracker)
					m.Vx, m.Vy, m.Vz = 0, 0, 0
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
			world.BroadcastPacket(packets.NewAddPassengerPacket(pl.GetEntityId(), -1))
			ridable.PassengerEntityId = -1
		} else {
			world.BroadcastPacket(packets.PlayerEntityMetadataPacketRiding(pl, true))
			world.BroadcastPacket(packets.NewAddPassengerPacket(pl.GetEntityId(), other.GetEntityId()))
			pl.IsRiding = other.GetEntityId()
			ridable.PassengerEntityId = pl.GetEntityId()
			pl.Lx = pl.X
			pl.Ly = pl.Y
			pl.Lz = pl.Z
		}
	}
}

func handlePlayerActionPacket(p packets.PlayerActionPacket, pl *player.Player, world *level.World) {
	if p.ActionId == 1 {
		world.MulticastPacket(packets.NewPlayerMetadataPacketSneak(pl, true), pl)
	}
	if p.ActionId == 2 {
		world.MulticastPacket(packets.NewPlayerMetadataPacketSneak(pl, false), pl)
	}
	if p.ActionId == 3 {
		world.RemoveSleeper(pl)
		p := packets.AnimationPacket{PlayerId: pl.GetEntityId(), Animation: 3}
		world.BroadcastPacket(p.Serialize())
	}
}

func BroadcastDespawn(world *level.World, id int32) {
	despawn := packets.DespawnEntityPacket{EntityId: id}
	world.BroadcastPacket(despawn.Serialize())
}
