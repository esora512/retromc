package packethandler

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"time"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/crafting"
	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

// World boundary constants (blocks). Chunks -1..0 on both axes → X/Z in [-16, 16).
const (
	worldMinX = -16.0
	worldMaxX = 17.0
	worldMinZ = -16.0
	worldMaxZ = 16.0
	ignoreY   = -999.0
)

type fluidPlacement struct {
	bucketId     int16
	isFluidBlock func(*constants.WBlock) bool
	newBlock     constants.WBlock
}

func handleUpdateSignPacket(p packets.UpdateSignPacket, world *level.World, pl *player.Player) {
	world.BroadcastPacket(p.Serialize())
}

func handleRespawnInPacket(connection net.Conn, p packets.RespawnPacket, world *level.World, pl *player.Player) {
	pl.X = player.SpawnX
	pl.Y = player.SpawnY
	pl.Z = player.SpawnZ
	pl.Stance = player.SpawnStance
	pl.Yaw = 0
	pl.Pitch = 0
	pl.OnGround = true
	pl.Immune = 0

	loc := int32(0)
	pl.SentChunks = make(player.ChunkSet)
	pl.HasInitializedChunks = false

	sendRespawn(connection, byte(loc))

	pl.SetHP(20)
	SendSetHealth(connection, 20.0)
	sendPlayerPositionAndLook(connection, 0, 0, 80)
	world.MulticastPacket(packets.NewAddPassengerPacket(pl.GetEntityId(), -1), pl)
}

func sendRespawn(connection net.Conn, world byte) {
	respawnPacket := packets.RespawnPacket{
		World: world,
	}
	connection.Write(respawnPacket.Serialize())
}

func SendSetHealth(connection net.Conn, health uint16) {
	setHealthPacket := packets.SetHealthPacket{
		Health: health,
	}
	connection.Write(setHealthPacket.Serialize())
}

func BroadcastPain(w *level.World, entityId int32) {
	p := packets.EntityEventPacket{
		EntityId: entityId,
		Action:   2,
	}
	w.BroadcastPacket(p.Serialize())
}

func BroadcastDeath(w *level.World, entityId int32) {
	p := packets.EntityEventPacket{
		EntityId: entityId,
		Action:   3,
	}
	w.BroadcastPacket(p.Serialize())
	log.Printf("Entity %d died", entityId)
}

func outOfBounds(x, z float64) bool {
	return x < worldMinX || x >= worldMaxX || z < worldMinZ || z >= worldMaxZ
}

// rubberBand sends the player back to their last valid position.
func rubberBand(connection net.Conn, pl *player.Player) {
	p := packets.PlayerPositionAndRotationPacket{
		X:        pl.X,
		Y:        pl.Y,
		Stance:   pl.Stance,
		Z:        pl.Z,
		Yaw:      pl.Yaw,
		Pitch:    pl.Pitch,
		OnGround: pl.OnGround,
	}
	connection.Write(p.Serialize())
}

// func logMovementDirection(pl *player.Player, newX, newZ float64) {
//  dx := newX - pl.X
//  dz := newZ - pl.Z

//  if math.Abs(dx) < 0.01 && math.Abs(dz) < 0.01 {
//      return // no meaningful movement, skip
//  }

//  var dirX, dirZ string
//  if dx > 0.01 {
//      dirX = "E"
//  } else if dx < -0.01 {
//      dirX = "W"
//  }
//  if dz > 0.01 {
//      dirZ = "S"
//  } else if dz < -0.01 {
//      dirZ = "N"
//  }

//  dir := dirX + dirZ
//  if dir == "" {
//      dir = "?"
//  }
//  log.Printf("[move] %s  dx=%.3f dz=%.3f", dir, dx, dz)
// }

func handlePlayerInputPacket(p packets.PlayerInputPacket, pl *player.Player, world *level.World) {
	log.Printf("Received PlayerInput packet: Strafe=%.2f Forward=%.2f Jump=%t Sneaking=%t",
		p.StrafeDirection, p.ForwardDirection, p.Jumping, p.Sneaking)
}

func applyFallDamage(world *level.World, pl *player.Player, newY float64, clientOnGround bool) {
	if pl.Immune < 200 {
		// 10s spawn immunity
		return
	}
	x := int32(math.Floor(pl.X))
	y := byte(math.Floor(newY - 0.01))
	z := int32(math.Floor(pl.Z))

	block := world.GetBlock(x, y, z, pl.Dimension)

	inWater := block.IsWater()
	onSolidGround := block.IsSolid() && !inWater

	diff := pl.Y - newY

	if !pl.OnGround && diff > 0 {
		pl.FallDistance += diff
	}

	if inWater {
		pl.FallDistance = 0
		pl.OnGround = false
		pl.Y = newY
		return
	}

	if onSolidGround && !pl.OnGround {
		if pl.FallDistance > 3 {
			dmg := int16(math.Ceil(pl.FallDistance - 3))

			newHP := pl.HP - dmg
			if newHP < 0 {
				newHP = 0
			}

			pl.SetHP(newHP)

			p := packets.SetHealthPacket{
				Health: uint16(pl.HP),
			}
			pl.Connection.Write(p.Serialize())

			if pl.HP == 0 {
				cMsgPkt := packets.ChatMessagePacket{
					Message: pl.GetName() + " was killed by gravity",
				}
				world.BroadcastPacket(cMsgPkt.Serialize())

				p := packets.EntityEventPacket{
					EntityId: pl.GetEntityId(),
					Action:   3,
				}
				world.BroadcastPacket(p.Serialize())
			}
		}

		pl.FallDistance = 0
	}

	pl.OnGround = onSolidGround
	pl.Y = newY
}

func handlePlayerPositionAndRotationPacket(connection net.Conn, p packets.PlayerPositionAndRotationPacket, pl *player.Player, world *level.World) {
	if p.X <= -1 && p.Y <= -1000000 && p.Z <= -1 {
		return
	}

	if pl.IsRiding != -1 {
		maybeRidable := world.Entities[pl.IsRiding]
		ridable, ok := maybeRidable.(*entities.RideableEntity)
		if ok && ridable.ObjectType == constants.ObjectBoat {
			if p.Y <= ignoreY {
				ridable.PassengerVelocityX = p.X
				ridable.PassengerVelocityZ = p.Z
			}
			// Always snap player to boat and update look
			//pl.Yaw = p.Yaw
			//pl.Pitch = p.Pitch
			//pl.Stance = p.Stance
			pl.X = ridable.X
			pl.Y = ridable.Y
			pl.Z = ridable.Z
			return
		}
	}

	x, y, z := p.X, p.Y, p.Z

	if p.Y <= ignoreY && pl.IsRiding != -1 {
		maybeRidable := world.Entities[pl.IsRiding]
		ridable, _ := maybeRidable.(*entities.RideableEntity)
		x, y, z = ridable.X, ridable.Y, ridable.Z
		updateChunks(world, x, z, pl)
		if p.Yaw == pl.Yaw && p.Pitch == pl.Pitch {
			return
		}
	}

	if y < 0 {
		pl.BelowZeroHeightCount++
		if pl.BelowZeroHeightCount > 10 {
			SendSetHealth(connection, 0)
			return
		}
	}
	if y >= 0 {
		pl.BelowZeroHeightCount = 0
	}

	pl.MovementState.X = x
	pl.MovementState.Y = y
	pl.MovementState.Z = z
	pl.MovementState.Yaw = p.Yaw
	pl.MovementState.Pitch = p.Pitch
	pl.MovementState.PositionAndRotationChanged = true
	pl.MovementState.UntrackPositionAndRotationIn = 10
	applyFallDamage(world, pl, p.Y, p.OnGround)

	pl.X = x
	pl.Y = y
	pl.Z = z
	pl.Stance = p.Stance
	pl.Yaw = p.Yaw
	pl.Pitch = p.Pitch
	pl.OnGround = p.OnGround
	updateChunks(world, x, z, pl)
}

func handlePlayerPositionPacket(connection net.Conn, p packets.PlayerPositionPacket, pl *player.Player, world *level.World) {
	if p.X <= -1 && p.Y <= -1000000 && p.Z <= -1 {
		return
	}

	if pl.IsRiding != -1 {
		maybeRidable := world.Entities[pl.IsRiding]
		ridable, ok := maybeRidable.(*entities.RideableEntity)
		if ok && ridable.ObjectType == constants.ObjectBoat {
			if p.Y <= ignoreY && ((p.X != pl.X || p.Z != pl.Z) && (p.X != 0 && p.Z != 0)) {
				ridable.PassengerVelocityX = p.X
				ridable.PassengerVelocityZ = p.Z
			}
			// Always snap player to boat
			pl.Stance = p.Stance
			pl.OnGround = p.OnGround
			pl.X = ridable.X
			pl.Y = ridable.Y
			pl.Z = ridable.Z
			updateChunks(world, pl.X, pl.Z, pl)
			return
		}
	}

	x, y, z := p.X, p.Y, p.Z

	if p.Y <= ignoreY && pl.IsRiding != -1 {
		maybeRidable := world.Entities[pl.IsRiding]
		ridable, ok := maybeRidable.(*entities.RideableEntity)
		if ok {
			x, y, z = ridable.X, ridable.Y, ridable.Z
		}
	}
	pl.MovementState.X = x
	pl.MovementState.Y = y
	pl.MovementState.Z = z
	pl.MovementState.PositionChanged = true
	pl.MovementState.UntrackPositionIn = 10
	applyFallDamage(world, pl, p.Y, p.OnGround)

	pl.X = x
	pl.Y = y
	pl.Z = z
	pl.Stance = p.Stance
	pl.OnGround = p.OnGround
	updateChunks(world, x, z, pl)

}

func handlePlayerRotationPacket(p packets.PlayerRotationPacket, pl *player.Player, world *level.World) {
	pl.MovementState.Yaw = p.Yaw
	pl.MovementState.Pitch = p.Pitch
	pl.MovementState.RotationChanged = true
	pl.MovementState.UntrackPositionAndRotationIn = 10

	pl.Yaw = p.Yaw
	pl.Pitch = p.Pitch
	pl.OnGround = p.OnGround
}

func High8Bits(n uint16) byte {
	return byte(n >> 8)
}

const dropInitVelocity = 0.3
const dropRandomVelocity = 0.02
const playerEyeHeight = 1.62
const JavaPI = 3.141592653589793

func DropItemFromPlayer(world *level.World, pl *player.Player, typeId int16, metadata uint16, count byte) {
	if count == 0 {
		return
	}

	yawRad := float64(pl.Yaw) * (JavaPI / 180)
	pitchRad := float64(pl.Pitch) * (JavaPI / 180)

	x := pl.X
	y := pl.Y + playerEyeHeight
	z := pl.Z

	velX := -math.Sin(yawRad) * math.Cos(pitchRad) * dropInitVelocity
	velZ := math.Cos(yawRad) * math.Cos(pitchRad) * dropInitVelocity
	velY := -math.Sin(pitchRad)*dropInitVelocity + 0.1

	angle := float64(rand.Float32()) * JavaPI * 2
	speed := float64(rand.Float32()) * dropRandomVelocity
	velX += math.Cos(angle) * speed
	velZ += math.Sin(angle) * speed
	velY += float64(rand.Float32()-rand.Float32()) * 0.1

	CreateDroppedItem(world, x, y, z, int32(typeId), count, byte(metadata), velX, velY, velZ, 45, pl.Dimension)
	sendEquipmentChangeForHotbarSlot(world, pl)
}

func handleMineBlockPacket(connection net.Conn, p packets.MineBlockPacket, world *level.World, pl *player.Player) {
	if p.Status == 4 {
		dropHeldItemStack(connection, world, pl)
		return
	}
	if pl.IsRiding != -1 {
		return
	}
	world.MulticastPacket(packets.ArmSwing(pl), pl)

	oldBlock := world.GetBlock(p.X, p.Y, p.Z, pl.Dimension)
	if !shouldProcessDigging(p, pl, oldBlock) {
		return
	}

	damageHeldItemOnDig(pl)

	if oldBlock.TypeId == 0x00 {
		return
	}

	removeMinedBlockEntity(world, p, oldBlock)

	air := constants.NewAirBlock()

	above := world.GetBlock(p.X, p.Y+1, p.Z, pl.Dimension)
	if above.IsSnowLayer() {
		world.SetBlockInQueue(p.X, int32(p.Y)+1, p.Z, air, pl.Dimension)
	}

	world.SetBlockInQueue(p.X, int32(p.Y), p.Z, air, pl.Dimension)
	world.TriggerFallableUpdate(p.X, int32(p.Y), p.Z, world.SetBlockInQueue, pl.Dimension)

	if oldBlock.TypeId == byte(constants.Bed.Value) {
		var hX, hZ int32
		for _, n := range level.GetNeighbours() {
			head := world.GetBlock(p.X+n.Dx, p.Y, p.Z+n.Dz, pl.Dimension)
			if head.TypeId == byte(constants.Bed.Value) {
				hX = p.X + n.Dx
				hZ = p.Z + n.Dz
				break
			}
		}
		world.SetBlockInQueue(p.X, int32(p.Y), p.Z, air, pl.Dimension)
		world.SetBlockInQueue(hX, int32(p.Y), hZ, air, pl.Dimension)
	}

	if oldBlock.TypeId == byte(constants.WoodenDoor.Value) || oldBlock.TypeId == byte(constants.IronDoor.Value) {
		target := oldBlock.TypeId
		var neighbours = []int{1, -1}
		var hY int32
		for _, n := range neighbours {
			topOrbot := world.GetBlock(p.X, p.Y+byte(n), p.Z, pl.Dimension)
			if topOrbot.TypeId == target {
				hY = int32(p.Y) + int32(n)
				break
			}
		}
		world.SetBlockInQueue(p.X, int32(p.Y), p.Z, air, pl.Dimension)
		world.SetBlockInQueue(p.X, hY, p.Z, air, pl.Dimension)
	}

	if oldBlock.TypeId == byte(constants.Log.Value) {
		world.TriggerLeafUpdate(p.X, int32(p.Y), p.Z, world.SetBlockInQueue, pl.Dimension)
	}

	blockItem, blockMeta, count := computeMinedDrop(world, p, oldBlock, pl)
	if blockItem == 0 {
		return
	}

	CreateAndSetMovementDroppedItem(world, float64(p.X), float64(p.Y), float64(p.Z), blockItem, blockMeta, count, pl.Dimension, 10)
	world.TriggerFluidUpdate(p.X, int32(p.Y), p.Z, world.SetBlockInQueue, pl.Dimension)
}

func dropHeldItemStack(connection net.Conn, world *level.World, pl *player.Player) {
	item := pl.Inventory.PeekItem(pl.HotbarSlot)
	if item.TypeId == -1 {
		return
	}
	typeId := item.TypeId
	metadata := item.Metadata
	pl.Inventory.RemoveOne(pl.HotbarSlot)
	SendSetSlot(connection, 0, pl.HotbarSlot, pl.Inventory.Items[pl.HotbarSlot])
	DropItemFromPlayer(world, pl, typeId, metadata, 1)
}

func shouldProcessDigging(p packets.MineBlockPacket, pl *player.Player, oldBlock constants.WBlock) bool {
	finishedDigging := p.Status == 2 || (pl.IsCreative && p.Status == 0)
	if pl.Inventory.Items[pl.HotbarSlot].IsShovel() && oldBlock.TypeId == byte(constants.SnowLayer.Value) {
		return true
	}
	if finishedDigging {
		return true
	}
	return oldBlock.TypeId == byte(constants.Wheat.Value) ||
		oldBlock.TypeId == byte(constants.Sugarcane.Value) ||
		oldBlock.TypeId == byte(constants.Cactus.Value) ||
		oldBlock.TypeId == byte(constants.Sapling.Value) ||
		oldBlock.TypeId == byte(constants.Torch.Value) ||
		oldBlock.TypeId == byte(constants.Dandelion.Value) ||
		oldBlock.TypeId == byte(constants.Rose.Value)
}

func damageHeldItemOnDig(pl *player.Player) {
	hotBarItem := pl.Inventory.Items[pl.HotbarSlot]
	if !crafting.HasDurability(hotBarItem.TypeId) {
		return
	}
	hotBarItem.Metadata += 1
	if hotBarItem.Metadata == crafting.Durability(hotBarItem.TypeId) {
		pl.Inventory.Items[pl.HotbarSlot] = inventory.EmptyItem()
		SendSetSlot(pl.Connection, 0, pl.HotbarSlot, inventory.EmptyItem())

	} else {
		pl.Inventory.Items[pl.HotbarSlot].Metadata += 1
		SendSetSlot(pl.Connection, 0, pl.HotbarSlot, hotBarItem)
	}
}

func removeMinedBlockEntity(world *level.World, p packets.MineBlockPacket, oldBlock constants.WBlock) {
	if oldBlock.TypeId == byte(constants.Chest.Value) {
		world.RemoveChest(p.X, int32(p.Y), p.Z)
	}

	if oldBlock.TypeId == byte(constants.Dispenser.Value) {
		world.RemoveDispenser(p.X, int32(p.Y), p.Z)
	}

	if oldBlock.TypeId == byte(constants.Furnace.Value) || oldBlock.TypeId == byte(constants.FurnaceLit.Value) {
		world.RemoveFurnace(p.X, int32(p.Y), p.Z)
	}
}

func computeMinedDrop(world *level.World, p packets.MineBlockPacket, oldBlock constants.WBlock, pl *player.Player) (blockItem int16, blockMeta byte, count byte) {
	count = 1
	blockItem = int16(oldBlock.TypeId)
	blockMeta = oldBlock.Metadata

	if blockItem == constants.Bed.Value {
		return constants.BedItem.Value, 0, 1
	}

	if blockItem == constants.Trapdoor.Value {
		return constants.Trapdoor.Value, 0, 1
	}

	if blockItem == constants.StoneButton.Value {
		return constants.StoneButton.Value, 0, 1
	}

	if blockItem == constants.CoalOre.Value {
		return constants.Coal.Value, 0, 1
	}

	if blockItem == constants.IronOre.Value {
		if pl.Inventory.Items[pl.HotbarSlot].TypeId != constants.IronPickaxe.Value &&
			pl.Inventory.Items[pl.HotbarSlot].TypeId != constants.DiamondPickaxe.Value &&
			pl.Inventory.Items[pl.HotbarSlot].TypeId != constants.GoldPickaxe.Value &&
			pl.Inventory.Items[pl.HotbarSlot].TypeId != constants.StonePickaxe.Value {
			return 0, 0, 0
		}
	}

	if blockItem == constants.RedstoneOreOff.Value {
		if pl.Inventory.Items[pl.HotbarSlot].TypeId != constants.IronPickaxe.Value &&
			pl.Inventory.Items[pl.HotbarSlot].TypeId != constants.DiamondPickaxe.Value &&
			pl.Inventory.Items[pl.HotbarSlot].TypeId != constants.GoldPickaxe.Value {
			return 0, 0, 0
		}
		return constants.Redstone.Value, 0, 1
	}

	if blockItem == constants.LapisLazuliOre.Value {
		roll := rand.Intn(6) + 1
		return constants.Dye.Value, 4, byte(roll)
	}

	if blockItem == constants.DiamondOre.Value {
		if pl.Inventory.Items[pl.HotbarSlot].TypeId != constants.IronPickaxe.Value &&
			pl.Inventory.Items[pl.HotbarSlot].TypeId != constants.DiamondPickaxe.Value &&
			pl.Inventory.Items[pl.HotbarSlot].TypeId != constants.GoldPickaxe.Value {
			return 0, 0, 0
		}
		return constants.Diamond.Value, 0, 1
	}

	if blockItem == constants.SnowLayer.Value {
		if pl.Inventory.Items[pl.HotbarSlot].IsShovel() {
			blockItem = constants.Snowball.Value
			count = 4
			return blockItem, 0, count
		} else {
			blockItem = 0
			return 0, 0, 0
		}
	}

	if blockItem == constants.Stone.Value || blockItem == constants.LavaStill.Value || blockItem == constants.LavaFlowing.Value {
		blockItem = constants.Cobblestone.Value
	}

	if blockItem == int16(constants.FurnaceLit.Value) {
		blockItem = int16(constants.Furnace.Value)
	}

	if blockItem == constants.SignGround.Value {
		blockItem = constants.Sign.Value
	}

	if blockItem == constants.Rail.Value || blockItem == constants.PoweredRail.Value || blockItem == constants.DetectorRail.Value {
		blockMeta = byte(0)
	}

	if blockItem == constants.Wheat.Value {
		if blockMeta < 7 {
			blockItem = constants.Seeds.Value
		} else {
			blockItem = constants.WheatItem.Value
		}
		bk := level.BlockKey{X: p.X, Y: p.Y, Z: p.Z}
		chunk := world.GetLoadedChunk(p.X, p.Z, pl.Dimension)
		logic := chunk.Logic
		delete(logic.Growables, bk)
	}

	if blockItem == constants.Leaves.Value {
		roll := rand.Intn(100)
		switch {
		case roll < 5: // 5% chance
			blockItem = constants.Apple.Value
		case roll < 15: // 10% chance (5–14)
			blockItem = constants.Sapling.Value
		default: // 85% chance — nothing drops
			blockItem = 0
		}
	}

	if blockItem == constants.Sugarcane.Value || blockItem == constants.Cactus.Value {
		if blockItem == constants.Sugarcane.Value {
			blockItem = constants.SugarcaneItem.Value
		}
		blockMeta = 0

		bk := level.BlockKey{X: p.X, Y: p.Y, Z: p.Z}
		chunk := world.GetLoadedChunk(p.X, p.Z, pl.Dimension)
		delete(chunk.Logic.Growables, bk)

		for i := 1; i <= 3; i++ {
			aboveY := p.Y + byte(i)
			above := world.GetBlock(p.X, aboveY, p.Z, pl.Dimension)
			if above.TypeId != oldBlock.TypeId {
				break
			}
			air := constants.NewAirBlock()
			world.SetBlockInQueue(p.X, int32(aboveY), p.Z, air, pl.Dimension)
			count++
		}
	}

	if blockItem == constants.Sapling.Value {
		chunk := world.GetLoadedChunk(p.X, p.Z, pl.Dimension)
		bk := level.BlockKey{X: p.X, Y: p.Y, Z: p.Z}
		delete(chunk.Logic.Growables, bk)
	}

	if blockItem == constants.Grass.Value {
		chunk := world.GetLoadedChunk(p.X, p.Z, pl.Dimension)
		blockItem = constants.Dirt.Value
		bk := level.BlockKey{X: p.X, Y: p.Y, Z: p.Z}
		delete(chunk.Logic.Growables, bk)
	}

	if blockItem == constants.WoodenDoor.Value {
		return constants.WoodenDoorItem.Value, 0, 1
	}

	if blockItem == constants.IronDoor.Value {
		return constants.IronDoor.Value, 0, 1
	}

	return blockItem, blockMeta, count
}

func CreateAndSetMovementDroppedItem(world *level.World, x, y, z float64, blockItem int16, blockMeta byte, count byte, dim, delay int32) {
	velX := float64(rand.Float32()-rand.Float32()) * 0.1
	velY := float64(rand.Float32()) * 0.2
	velZ := float64(rand.Float32()-rand.Float32()) * 0.1
	CreateDroppedItem(world, x, y, z, int32(blockItem), count, blockMeta, velX, velY, velZ, delay, dim)
}

func raycastForWater(world *level.World, pl *player.Player, maxDistance float64) (int, int, int, bool) {
	const step = 0.1
	const eyeHeight = 1.62

	yawRad := float64(pl.Yaw) * math.Pi / 180
	pitchRad := float64(pl.Pitch) * math.Pi / 180

	dx := -math.Sin(yawRad) * math.Cos(pitchRad)
	dy := -math.Sin(pitchRad)
	dz := math.Cos(yawRad) * math.Cos(pitchRad)

	ox := float64(pl.X)
	oy := float64(pl.Y) + eyeHeight
	oz := float64(pl.Z)

	lastX, lastY, lastZ := math.MinInt32, math.MinInt32, math.MinInt32

	for d := 0.0; d <= maxDistance; d += step {
		px := ox + dx*d
		py := oy + dy*d
		pz := oz + dz*d

		bx := int(math.Floor(px))
		by := int(math.Floor(py))
		bz := int(math.Floor(pz))

		if bx == lastX && by == lastY && bz == lastZ {
			continue
		}
		lastX, lastY, lastZ = bx, by, bz

		if by < 0 || by >= level.CHUNK_SIZE_Y {
			continue
		}

		block := world.GetBlock(int32(bx), byte(by), int32(bz), pl.Dimension)

		if block.IsWater() {
			return bx, by, bz, true
		}
	}

	return 0, 0, 0, false
}

func tryPlaceBoatNoTarget(connection net.Conn, world *level.World, pl *player.Player) bool {
	x, y, z, found := raycastForWater(world, pl, 8.0)
	if !found {
		return false
	}

	slot := pl.HotbarSlot
	tryPlaceBoat(connection, world, pl, int32(x), y, int32(z), slot)
	return true
}

func interactWithBed(oldExisting *constants.WBlock, world *level.World, pl *player.Player, p packets.PlaceBlockPacket) bool {
	if oldExisting.IsBed() {
		var hX, hZ int32
		if oldExisting.IsBedHead() {
			hX, hZ = p.X, p.Z
		} else {
			for _, n := range level.GetNeighbours() {
				head := world.GetBlock(p.X+n.Dx, p.Y, p.Z+n.Dz, pl.Dimension)
				if head.IsBedHead() {
					hX, hZ = p.X+n.Dx, p.Z+n.Dz
				}
			}
		}

		if world.IsNight() {
			p := packets.NewInteractWithBlockPacket(pl.GetEntityId(), 0, hX, p.Y, hZ)
			mP := packets.AnimationPacket{PlayerId: pl.GetEntityId(), Animation: 1}
			pl.Connection.Write(p)
			world.MulticastPacket(mP.Serialize(), pl)
			world.AddSleeper(pl)
			go func() { time.Sleep(time.Millisecond * 500); world.MulticastPacket(p, pl) }()
		} else {
			sendDebugMessage(pl, fmt.Sprintln("Can only sleep at night..."))
		}
		return true
	}
	return false
}

type AABB struct {
	MinX, MinY, MinZ float64
	MaxX, MaxY, MaxZ float64
}

func (a AABB) Intersects(b AABB) bool {
	return a.MinX < b.MaxX && a.MaxX > b.MinX &&
		a.MinY < b.MaxY && a.MaxY > b.MinY &&
		a.MinZ < b.MaxZ && a.MaxZ > b.MinZ
}

func playerAABB(pl *player.Player) AABB {
	const halfWidth = 0.3
	const height = 1.8
	return AABB{
		MinX: pl.X - halfWidth,
		MaxX: pl.X + halfWidth,
		MinY: pl.Y,
		MaxY: pl.Y + height,
		MinZ: pl.Z - halfWidth,
		MaxZ: pl.Z + halfWidth,
	}
}

func blockAABB(x, y, z int, margin float64) AABB {
	return AABB{
		MinX: float64(x) - margin,
		MaxX: float64(x+1) + margin,
		MinY: float64(y) - margin,
		MaxY: float64(y+1) + margin,
		MinZ: float64(z) - margin,
		MaxZ: float64(z+1) + margin,
	}
}

func placementCollidesWithPlayer(pl *player.Player, x, y, z int32) bool {
	block := blockAABB(int(x), int(y), int(z), 0.05)
	player := playerAABB(pl)
	return block.Intersects(player)
}

func handlePlaceBlockPacket(connection net.Conn, p packets.PlaceBlockPacket, world *level.World, pl *player.Player) {
	oldExisting := world.GetBlock(p.X, byte(p.Y), p.Z, pl.Dimension)
	logPlacementDebug(pl, oldExisting)

	if oldExisting.IsSnowLayer() {
		return
	}

	if openBlockEntityUI(connection, world, pl, p, oldExisting) {
		return
	}

	if interactWithBed(&oldExisting, world, pl, p) {
		return
	}

	if oldExisting.IsTrapdoor() {
		interactWithTrapDoor(world, p.X, int32(p.Y), p.Z, pl.Dimension)
		return
	}

	if oldExisting.IsDoor() {
		interactWithDoor(world, p.X, int32(p.Y), p.Z, pl.Dimension)
		return
	}

	heldItem := pl.Inventory.PeekItem(pl.HotbarSlot)
	if p.X == -1 && p.Y == 255 && p.Z == -1 && heldItem.TypeId == constants.Boat.Value {
		//log.Printf("Player Looks At: x=%f, y=%f, z=%f, yaw=%f, pitch=%f", pl.X, pl.Y, pl.Z, pl.Yaw, pl.Pitch)
		if tryPlaceBoatNoTarget(connection, world, pl) {
			return
		}
		return
	}

	if !canPlaceHeldItem(heldItem) {
		// Only place blocks if block is in hotbar slot
		return
	}

	if heldItem.TypeId == constants.FlintAndSteel.Value {
		handleFlintAndSteelPlacement(world, p, pl, heldItem)
		return
	}

	if tryTillSoil(world, p, oldExisting, heldItem, pl.Dimension) {
		return
	}

	pl.HotbarLocked.Store(true)
	defer pl.HotbarLocked.Store(false)
	// X/Y/Z are the clicked block; the new block goes on the adjacent face.
	// Face: 0=-Y  1=+Y  2=-Z  3=+Z  4=-X  5=+X
	newX, newY, newZ := placementTargetCoords(p, world, pl.Dimension)

	// Reject out-of-bounds Y.
	if newY < 0 || newY >= level.CHUNK_SIZE_Y {
		return
	}

	if placementCollidesWithPlayer(pl, newX, int32(newY), newZ) {
		return
	}

	// Reject placement into a chunk that was never sent to the client.
	// cx := level.WorldToChunkCoord(newX)
	// cz := level.WorldToChunkCoord(newZ)
	// if !world.ChunkExists(cx, cz) {
	//  return
	// }

	// Only place into air — don't overwrite existing blocks.
	existing := world.GetBlock(newX, byte(newY), newZ, pl.Dimension)
	if !existing.IsAir() && !existing.IsLiquid() && !existing.IsSnowLayer() {
		return
	}

	if heldItem.TypeId == constants.BedItem.Value {
		handleBedPlacement(world, p, pl)
		return
	}

	if heldItem.TypeId == constants.Trapdoor.Value {
		handleTrapDoor(world, p, pl, newX, int32(newY), newZ)
		return
	}

	if heldItem.TypeId == constants.WoodenDoorItem.Value || heldItem.TypeId == constants.IronDoorItem.Value {
		handleDoorPlacement(world, p, pl, int32(heldItem.TypeId), int32(p.Face))
		return
	}

	if tryPlacePlant(connection, world, pl, newX, newY, newZ, oldExisting, heldItem) {
		return
	}

	if tryScoopFluidWithBucket(connection, world, pl, newX, newY, newZ, existing, heldItem) {
		return
	}

	// Verify the player actually has the item they're trying to place.
	//slot := pl.Inventory.FindFirstSlotWith(p.ItemId)
	slot := pl.HotbarSlot
	item := pl.Inventory.PeekItem(slot)
	block := constants.NewBlockById(p.ItemId, byte(item.Metadata))
	//log.Printf("Placing block: TypeId=%d Meta=%d at (%d, %d, %d)", block.TypeId, block.Metadata, newX, newY, newZ)
	if heldItem.TypeId == -1 {
		return
	}

	if block.IsRail() {
		placeRailBlock(world, &block, newX, newY, newZ, pl.Dimension)
		pl.Inventory.RemoveOne(slot)
		return
	}

	if block.IsDirectional() {
		if !configureDirectionalBlock(world, pl, &block, newX, newY, newZ, heldItem, p) {
			return
		}
	}

	// Handle minecart placement
	if heldItem.TypeId == constants.Minecart.Value {
		tryPlaceMinecart(connection, world, pl, newX, newY, newZ, slot)
		return
	}

	if heldItem.TypeId == constants.Boat.Value {
		tryPlaceBoat(connection, world, pl, newX, newY, newZ, slot)
		return
	}

	if tryPlaceFluidFromBucket(connection, world, pl, block, newX, newY, newZ, heldItem, slot) {
		return
	}

	finalizePlacement(connection, world, pl, block, newX, newY, newZ, p, slot)
}

func logPlacementDebug(pl *player.Player, oldExisting constants.WBlock) {
	if !pl.DebugBlock {
		return
	}
	sendDebugMessage(pl, fmt.Sprintf("Block type=%d meta=%d, light=%d, skylight=%d", oldExisting.TypeId, oldExisting.Metadata, oldExisting.Light, oldExisting.SkyLight))
}

func openBlockEntityUI(connection net.Conn, world *level.World, pl *player.Player, p packets.PlaceBlockPacket, oldExisting constants.WBlock) bool {
	if oldExisting.TypeId == byte(constants.CraftingTable.Value) {
		cp := packets.NewCraftingTable()
		connection.Write(cp.Serialize())
		pl.InventoryType = player.WorkbenchInventory
		return true
	}

	if oldExisting.TypeId == byte(constants.Chest.Value) {
		chest := world.GetChest(p.X, int32(p.Y), p.Z)
		chestPacket := packets.NewChest(byte(chest.Size))
		connection.Write(chestPacket.Serialize())
		pl.InventoryType = player.ChestInventory
		pl.Chest.X = int32(p.X)
		pl.Chest.Y = int32(p.Y)
		pl.Chest.Z = int32(p.Z)
		sendChestContents(connection, chest)
		return true
	}

	if oldExisting.TypeId == byte(constants.Dispenser.Value) {
		dispenser := world.GetDispenser(p.X, int32(p.Y), p.Z)
		if dispenser == nil {
			return true
		}
		dispenserPacket := packets.NewDispenser()
		connection.Write(dispenserPacket.Serialize())
		pl.InventoryType = player.DispenserInventory
		pl.Dispenser.X = int32(p.X)
		pl.Dispenser.Y = int32(p.Y)
		pl.Dispenser.Z = int32(p.Z)
		sendDispenserContents(connection, dispenser)
		return true
	}

	if oldExisting.TypeId == byte(constants.Furnace.Value) || oldExisting.TypeId == byte(constants.FurnaceLit.Value) {
		furnace := world.GetFurnace(p.X, int32(p.Y), p.Z)
		if furnace == nil {
			return true
		}
		furnacePacket := packets.NewFurnace()
		connection.Write(furnacePacket.Serialize())
		pl.InventoryType = player.FurnaceInventory
		pl.Furnace.X = int32(p.X)
		pl.Furnace.Y = int32(p.Y)
		pl.Furnace.Z = int32(p.Z)
		sendFurnaceContents(connection, furnace)
		return true
	}

	return false
}

func canPlaceHeldItem(heldItem inventory.Item) bool {
	if heldItem.TypeId > 96 &&
		heldItem.TypeId != constants.Boat.Value &&
		heldItem.TypeId != constants.Minecart.Value &&
		heldItem.TypeId != constants.Sign.Value &&
		heldItem.TypeId != constants.WaterBucket.Value &&
		heldItem.TypeId != constants.Bucket.Value &&
		heldItem.TypeId != constants.LavaBucket.Value &&
		!heldItem.IsHoe() &&
		heldItem.TypeId != constants.Seeds.Value &&
		heldItem.TypeId != constants.SugarcaneItem.Value &&
		heldItem.TypeId != constants.Sapling.Value &&
		heldItem.TypeId != constants.FlintAndSteel.Value &&
		heldItem.TypeId != constants.BedItem.Value &&
		heldItem.TypeId != constants.IronDoorItem.Value &&
		heldItem.TypeId != constants.WoodenDoorItem.Value {
		return false
	}
	return true
}

type Facing byte

const (
	FacingNorth Facing = iota // -X
	FacingSouth               // +X
	FacingEast                // +Z
	FacingWest                // -Z
)

func facingFromFace(face int32) Facing {
	switch face {
	case 3:
		log.Println("West")
		return FacingWest
	case 2:
		log.Println("East")
		return FacingEast
	case 4:
		log.Println("North")
		return FacingNorth
	case 5:
		log.Println("South")
		return FacingSouth
	default:
		return FacingSouth
	}
}

func trapDoorMeta(open bool, facing Facing) byte {
	if open {
		return trapDoorOpen[facing]
	} else {
		return trapDoorClosed[facing]
	}
}

func doorMeta(top, open bool, facing Facing) byte {
	switch {
	case !top && !open:
		return bottomClosedByFacing[facing]
	case !top && open:
		return bottomOpenByFacing[facing]
	case top && !open:
		return topClosedByFacing[facing]
	default:
		return topOpenByFacing[facing]
	}
}

type doorState struct {
	top    bool
	open   bool
	facing Facing
}

type trapDoorState struct {
	open   bool
	facing Facing
}

var trapDoorClosed = map[Facing]byte{
	FacingEast:  0,
	FacingWest:  1,
	FacingNorth: 2,
	FacingSouth: 3,
}

var trapDoorOpen = map[Facing]byte{
	FacingEast:  4,
	FacingWest:  5,
	FacingNorth: 6,
	FacingSouth: 7,
}

var trapDoorStates = [8]trapDoorState{
	{false, FacingEast},
	{false, FacingWest},
	{false, FacingNorth},
	{false, FacingSouth},

	{true, FacingEast},
	{true, FacingWest},
	{true, FacingNorth},
	{true, FacingSouth},
}

var bottomClosedByFacing = map[Facing]byte{
	FacingSouth: 2, // +X
	FacingEast:  1, // +Z
	FacingNorth: 0, // -X
	FacingWest:  3, // -Z
}

var bottomOpenByFacing = map[Facing]byte{
	FacingSouth: 6, // +X
	FacingEast:  5, // +Z
	FacingNorth: 4, // -X
	FacingWest:  7, // -Z
}

var topClosedByFacing = map[Facing]byte{
	FacingSouth: 10, // +X
	FacingEast:  9,  // +Z
	FacingNorth: 8,  // -X
	FacingWest:  11, // -Z
}

var topOpenByFacing = map[Facing]byte{
	FacingSouth: 14, // +X
	FacingEast:  13, // +Z
	FacingNorth: 12, // -X
	FacingWest:  15, // -Z
}

var doorStates = [16]doorState{
	{false, false, FacingNorth}, // 0
	{false, false, FacingEast},  // 1
	{false, false, FacingSouth}, // 2
	{false, false, FacingWest},  // 3

	{false, true, FacingNorth}, // 4
	{false, true, FacingEast},  // 5
	{false, true, FacingSouth}, // 6
	{false, true, FacingWest},  // 7

	{true, false, FacingNorth}, // 8
	{true, false, FacingEast},  // 9
	{true, false, FacingSouth}, // 10
	{true, false, FacingWest},  // 11

	{true, true, FacingNorth}, // 12
	{true, true, FacingEast},  // 13
	{true, true, FacingSouth}, // 14
	{true, true, FacingWest},  // 15
}

func handleTrapDoor(world *level.World, p packets.PlaceBlockPacket, pl *player.Player, x, y, z int32) {
	if p.Face == 1 {
		return
	}
	face := int32(p.Face)
	trapDoor := constants.NewTrapdoorBlock(0)
	facing := facingFromFace(face)
	trapDoor.Metadata = trapDoorMeta(false, facing)
	world.SetBlockInQueue(x, y, z, trapDoor, pl.Dimension)

	pl.Inventory.RemoveOne(pl.HotbarSlot)
	SendSetSlot(pl.Connection, 0, pl.HotbarSlot, pl.Inventory.Items[pl.HotbarSlot])
	sendEquipmentChangeForHotbarSlot(world, pl)
}

func handleDoorPlacement(world *level.World, p packets.PlaceBlockPacket, pl *player.Player, typeId int32, face int32) {
	face = int32(yawToFace(pl.Yaw))
	var y int32 = int32(p.Y + 1)
	maybeSnow := world.GetBlock(p.X, p.Y, p.Z, pl.Dimension)
	if maybeSnow.IsSnowLayer() {
		y = int32(p.Y)
	}

	var bottomDoor, topDoor constants.WBlock
	if typeId == int32(constants.WoodenDoorItem.Value) {
		bottomDoor = constants.NewWoodenDoorBlock()
		topDoor = constants.NewWoodenDoorBlock()
	} else {
		bottomDoor = constants.NewIronDoorBlock()
		topDoor = constants.NewIronDoorBlock()
	}

	facing := facingFromFace(face)
	bottomDoor.Metadata = doorMeta(false, false, facing)
	topDoor.Metadata = doorMeta(true, false, facing)

	world.SetBlockInQueue(p.X, y, p.Z, bottomDoor, pl.Dimension)
	world.SetBlockInQueue(p.X, y+1, p.Z, topDoor, pl.Dimension)

	pl.Inventory.RemoveOne(pl.HotbarSlot)
	SendSetSlot(pl.Connection, 0, pl.HotbarSlot, pl.Inventory.Items[pl.HotbarSlot])
	sendEquipmentChangeForHotbarSlot(world, pl)
}

func interactWithTrapDoor(world *level.World, x, y, z int32, dimension int32) {
	block := world.GetBlock(x, byte(y), z, dimension)
	state := trapDoorStates[block.Metadata]

	newOpen := !state.open
	block.Metadata = trapDoorMeta(newOpen, state.facing)
	world.SetBlockInQueue(x, y, z, block, dimension)

}

func interactWithDoor(world *level.World, x, y, z int32, dimension int32) {
	block := world.GetBlock(x, byte(y), z, dimension)
	if int(block.Metadata) >= len(doorStates) {
		return
	}
	state := doorStates[block.Metadata]

	otherY := y + 1
	if state.top {
		otherY = y - 1
	}
	other := world.GetBlock(x, byte(otherY), z, dimension)
	otherState := doorStates[other.Metadata]

	newOpen := !state.open

	block.Metadata = doorMeta(state.top, newOpen, state.facing)
	other.Metadata = doorMeta(otherState.top, newOpen, otherState.facing)

	world.SetBlockInQueue(x, y, z, block, dimension)
	world.SetBlockInQueue(x, otherY, z, other, dimension)
}

func handleBedPlacement(world *level.World, p packets.PlaceBlockPacket, pl *player.Player) {
	var face byte = p.Face
	if face == 1 {
		face = yawToFace(pl.Yaw)
	}

	bed := constants.NewBedBlock(0)
	directions := bed.GetDirections()
	var headDir byte
	var dx, dz int32
	switch face {
	case 3:
		// West
		headDir = directions.West
		dx, dz = 0, -1
	case 2:
		// East
		headDir = directions.East
		dx, dz = 0, 1
	case 4:
		// North
		headDir = directions.North
		dx, dz = 1, 0
	case 5:
		// South
		headDir = directions.South
		dx, dz = -1, 0
	default:
		headDir = directions.South
		dx, dz = 0, 1
	}

	footBlock := constants.NewBedBlock(headDir)
	headBlock := constants.NewBedBlock(headDir | 0x8)

	var y int32 = int32(p.Y + 1)

	maybeSnow := world.GetBlock(p.X, p.Y, p.Z, pl.Dimension)
	if maybeSnow.IsSnowLayer() {
		y = int32(p.Y)
	}

	world.SetBlockInQueue(p.X, y, p.Z, footBlock, pl.Dimension)
	world.SetBlockInQueue(p.X+dx, y, p.Z+dz, headBlock, pl.Dimension)

	pl.Inventory.Items[pl.HotbarSlot] = inventory.EmptyItem()
	SendSetSlot(pl.Connection, 0, pl.HotbarSlot, inventory.EmptyItem())
	sendEquipmentChangeForHotbarSlot(world, pl)
}

func handleFlintAndSteelPlacement(world *level.World, p packets.PlaceBlockPacket, pl *player.Player, heldItem inventory.Item) {
	fire := constants.NewFireBlock()
	world.SetBlockInQueue(p.X, int32(p.Y+1), p.Z, fire, pl.Dimension)
	if !crafting.HasDurability(heldItem.TypeId) {
		return
	}
	heldItem.Metadata += 1
	if heldItem.Metadata == crafting.Durability(heldItem.TypeId) {
		pl.Inventory.Items[pl.HotbarSlot] = inventory.EmptyItem()
		SendSetSlot(pl.Connection, 0, pl.HotbarSlot, inventory.EmptyItem())

	} else {
		pl.Inventory.Items[pl.HotbarSlot].Metadata += 1
		SendSetSlot(pl.Connection, 0, pl.HotbarSlot, heldItem)
	}
}

func tryTillSoil(world *level.World, p packets.PlaceBlockPacket, oldExisting constants.WBlock, heldItem inventory.Item, dim int32) bool {
	if (oldExisting.TypeId != byte(constants.Dirt.Value) && oldExisting.TypeId != byte(constants.Grass.Value)) || !heldItem.IsHoe() {
		return false
	}
	tilled := constants.NewBlockById(constants.Farmland.Value, 0)
	world.SetBlockInQueue(p.X, int32(p.Y), p.Z, tilled, dim)
	return true
}

// Face: 0=-Y  1=+Y  2=-Z  3=+Z  4=-X  5=+X
func placementTargetCoords(p packets.PlaceBlockPacket, w *level.World, dim int32) (int32, int, int32) {
	existing := w.GetBlock(p.X, byte(p.Y), p.Z, dim)
	if existing.TypeId == byte(constants.SnowLayer.Value) {
		return p.X, int(p.Y), p.Z
	}
	newX, newY, newZ := p.X, int(p.Y), p.Z

	switch p.Face {
	case 0:
		newY--
	case 1:
		newY++
	case 2:
		newZ--
	case 3:
		newZ++
	case 4:
		newX--
	case 5:
		newX++
	}

	return newX, newY, newZ
}

func tryPlacePlant(connection net.Conn, world *level.World, pl *player.Player, newX int32, newY int, newZ int32, oldExisting constants.WBlock, heldItem inventory.Item) bool {
	rule, ok := level.PlantRules[heldItem.TypeId]
	if !ok {
		return false
	}
	if !rule.ValidGround(oldExisting.TypeId) {
		return true
	}
	meta := byte(0)
	if rule.UseMeta {
		meta = byte(heldItem.Metadata)
	}
	growable := level.PlantGrowable(world, rule.PlantedBlock, newX, byte(newY), newZ, meta, pl.Dimension)
	blockChange := packets.SetBlockPacket{
		X:         newX,
		Y:         byte(newY),
		Z:         newZ,
		BlockType: growable.TypeId,
		BlockMeta: growable.Metadata,
	}
	world.BroadcastPacket(blockChange.Serialize())
	pl.Inventory.RemoveOne(pl.HotbarSlot)
	SendSetSlot(connection, 0, pl.HotbarSlot, pl.Inventory.Items[pl.HotbarSlot])
	return true
}

func tryScoopFluidWithBucket(connection net.Conn, world *level.World, pl *player.Player, newX int32, newY int, newZ int32, existing constants.WBlock, heldItem inventory.Item) bool {
	if !existing.IsLiquid() || heldItem.TypeId != constants.Bucket.Value {
		return false
	}
	air := constants.NewAirBlock()
	world.SetBlockInQueue(newX, int32(newY), newZ, air, pl.Dimension)
	world.TriggerFluidUpdate(newX, int32(newY), newZ, world.SetBlockInQueue, pl.Dimension)
	var bucketItem inventory.Item
	if existing.IsWater() {
		bucketItem = inventory.Item{TypeId: constants.WaterBucket.Value, Count: 1}
	} else {
		bucketItem = inventory.Item{TypeId: constants.LavaBucket.Value, Count: 1}
	}
	pl.Inventory.Items[pl.HotbarSlot] = bucketItem
	SendSetSlot(connection, 0, pl.HotbarSlot, bucketItem)
	return true
}

func placeRailBlock(world *level.World, block *constants.WBlock, newX int32, newY int, newZ int32, dim int32) {
	railIds := map[byte]bool{
		byte(constants.Rail.Value):         true,
		byte(constants.PoweredRail.Value):  true,
		byte(constants.DetectorRail.Value): true,
	}

	hasRail := func(x, y, z int32) bool {
		b := world.GetBlock(x, byte(y), z, dim)
		return railIds[b.TypeId]
	}

	// computeMeta based on neighbouring rails:
	computeMeta := func(x, y, z int32) byte {
		north := hasRail(x, y, z-1)
		south := hasRail(x, y, z+1)
		east := hasRail(x+1, y, z)
		west := hasRail(x-1, y, z)

		northUp := hasRail(x, y+1, z-1)
		southUp := hasRail(x, y+1, z+1)
		eastUp := hasRail(x+1, y+1, z)
		westUp := hasRail(x-1, y+1, z)

		switch {
		case eastUp:
			return 2
		case westUp:
			return 3
		case northUp:
			return 4
		case southUp:
			return 5
		case east && south:
			return 6
		case west && south:
			return 7
		case west && north:
			return 8
		case east && north:
			return 9
		case east || west:
			return 1
		default:
			return 0
		}
	}

	x, y, z := int32(newX), int32(newY), int32(newZ)

	// Place the new rail with computed metadata
	block.Metadata = computeMeta(x, y, z)
	world.SetBlockInQueue(x, y, z, *block, dim)

	// Recalc each flat neighbour now that the new rail exists in the world
	recalcRail := func(nx, ny, nz int32) {
		existing := world.GetBlock(nx, byte(ny), nz, dim)
		if !railIds[existing.TypeId] {
			return
		}
		newMeta := computeMeta(nx, ny, nz)
		if newMeta == existing.Metadata {
			return
		}
		existing.Metadata = newMeta
		world.SetBlockInQueue(nx, ny, nz, existing, dim)
	}

	recalcRail(x, y, z-1) // north
	recalcRail(x, y, z+1) // south
	recalcRail(x+1, y, z) // east
	recalcRail(x-1, y, z) // west
}

func configureDirectionalBlock(world *level.World, pl *player.Player, block *constants.WBlock, newX int32, newY int, newZ int32, heldItem inventory.Item, p packets.PlaceBlockPacket) bool {
	if block.TypeId == byte(constants.Chest.Value) {
		check := world.PlaceChest(int32(newX), int32(newY), int32(newZ))
		if !check {
			return false
		}
	}

	if block.TypeId == byte(constants.Dispenser.Value) {
		check := world.PlaceDispenser(int32(newX), int32(newY), int32(newZ))
		if !check {
			return false
		}
	}

	if block.TypeId == byte(constants.Furnace.Value) {
		check := world.PlaceFurnace(int32(newX), int32(newY), int32(newZ))
		if !check {
			return false
		}
	}

	if heldItem.TypeId == constants.Sign.Value {
		block.TypeId = byte(constants.SignGround.Value)
	}

	directions := block.GetDirections()
	var face byte
	face = p.Face
	// TODO: Remember for which blocks face=1 is spammed; furnace was one of them...
	if face == 1 && block.TypeId != byte(constants.Torch.Value) {
		face = yawToFace(pl.Yaw)
	}
	switch face {
	case 3:
		// West
		block.Metadata = directions.West
		log.Println("West")
	case 2:
		// East
		block.Metadata = directions.East
		log.Println("East")
	case 4:
		// North
		block.Metadata = directions.North
		log.Println("North")
	case 5:
		// South
		block.Metadata = directions.South
		log.Println("South")
	default:
		block.Metadata = 0
	}

	return true
}

func yawToFace(yaw float32) byte {
	y := math.Mod(float64(yaw), 360)
	if y < 0 {
		y += 360
	}

	switch {
	case y >= 321 || y < 48:
		return 2 // East
	case y < 137:
		return 5 // South
	case y < 230:
		return 3 // West
	default:
		return 4 // North
	}
}

func tryPlaceMinecart(connection net.Conn, world *level.World, pl *player.Player, newX int32, newY int, newZ int32, slot int16) {
	beneath := world.GetBlock(newX, byte(newY-1), newZ, pl.Dimension)
	isRail := beneath.TypeId == byte(constants.Rail.Value) ||
		beneath.TypeId == byte(constants.PoweredRail.Value) ||
		beneath.TypeId == byte(constants.DetectorRail.Value)
	if !isRail {
		return
	}
	entityId := world.NextEntityId()
	world.AddRidable(entityId, pl.GetEntityId(), float64(newX), float64(newY), float64(newZ), 0, 0, 0, 10, pl.Dimension)
	pl.Inventory.RemoveOne(slot)
	SendSetSlot(connection, 0, slot, pl.Inventory.Items[slot])
	if pl.Inventory.PeekItem(slot).TypeId == -1 {
		sendEquipmentChangeForHotbarSlot(world, pl)
	}
}

func tryPlaceBoat(connection net.Conn, world *level.World, pl *player.Player, newX int32, newY int, newZ int32, slot int16) {
	entityId := world.NextEntityId()
	// Lift posY by BoatYOffset so the hitbox bottom sits on the block top
	// instead of half-burying the model in the block below.
	spawnY := float64(newY) + entities.BoatYOffset
	world.AddRidable(entityId, pl.GetEntityId(), float64(newX), spawnY, float64(newZ), 0, 0, 0, 1, pl.Dimension)
	pl.Inventory.RemoveOne(slot)
	SendSetSlot(connection, 0, slot, pl.Inventory.Items[slot])
	if pl.Inventory.PeekItem(slot).TypeId == -1 {
		sendEquipmentChangeForHotbarSlot(world, pl)
	}
}

func tryPlaceFluidFromBucket(connection net.Conn, world *level.World, pl *player.Player, block constants.WBlock, newX int32, newY int, newZ int32, heldItem inventory.Item, slot int16) bool {
	for _, fp := range []fluidPlacement{
		{constants.WaterBucket.Value, func(b *constants.WBlock) bool { return b.IsWater() }, constants.NewStillWaterBlock(0)},
		{constants.LavaBucket.Value, func(b *constants.WBlock) bool { return b.IsLava() }, constants.NewStillLavaBlock(0)},
	} {
		if !fp.isFluidBlock(&block) && heldItem.TypeId != fp.bucketId {
			continue
		}
		b := fp.newBlock
		world.SetBlockInQueue(newX, int32(newY), newZ, b, pl.Dimension)
		if heldItem.TypeId == fp.bucketId {
			bucket := inventory.Item{TypeId: constants.Bucket.Value, Count: 1, Metadata: 0}
			pl.Inventory.Items[slot] = bucket
			SendSetSlot(connection, 0, slot, bucket)
		} else {
			pl.Inventory.RemoveOne(slot)
			SendSetSlot(connection, 0, slot, pl.Inventory.Items[slot])
		}
		world.TriggerFluidUpdate(newX, int32(newY), newZ, world.SetBlockInQueue, pl.Dimension)
		return true
	}
	return false
}

func finalizePlacement(connection net.Conn, world *level.World, pl *player.Player, block constants.WBlock, newX int32, newY int, newZ int32, p packets.PlaceBlockPacket, slot int16) {
	world.SetBlockInQueue(newX, int32(newY), newZ, block, pl.Dimension)
	world.TriggerFluidUpdate(newX, int32(newY), newZ, world.SetBlockInQueue, pl.Dimension)
	world.TriggerFallableUpdate(p.X, int32(p.Y), p.Z, world.SetBlockInQueue, pl.Dimension)

	// Decrement the item in the in-memory inventory and sync to client.
	pl.Inventory.RemoveOne(slot)
	SendSetSlot(connection, 0, slot, pl.Inventory.Items[slot])
	if pl.Inventory.PeekItem(slot).TypeId == -1 {
		sendEquipmentChangeForHotbarSlot(world, pl)
	}

	cx := level.WorldToChunkCoord(int32(newX))
	cz := level.WorldToChunkCoord(int32(newZ))
	coord := level.ChunkCoord{X: cx, Z: cz}
	if !pl.SentChunks.Has(coord.String()) {
		chunk := world.GetOrCreateChunk(cx, cz, pl.Dimension)
		pre := packets.SetChunkVisibilityPacket{X: cx, Z: cz, Mode: true}
		connection.Write(pre.Serialize())
		mapChunk := packets.ChunkBlockRegionPacket{}
		mapChunk.Apply(*chunk)
		connection.Write(mapChunk.Serialize())
		pl.SentChunks.Set(coord.String(), coord.X, coord.Z)
	}
}

func handleSetHotbarSlot(p packets.SetHotbarSlotPacket, pl *player.Player, world *level.World) {
	// Drop the update while a BlockPlacement is in progress to avoid a race
	// where a slot change arriving just after placement resets the wrong slot.
	if pl.HotbarLocked.Load() {
		return
	}
	pl.HotbarSlot = p.Slot + 36
	sendEquipmentChangeForHotbarSlot(world, pl)
}
