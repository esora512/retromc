package packethandler

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"

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
	isFluidBlock func(*level.Block) bool
	newBlock     level.Block
}

func handleSignUpdateInPacket(p packets.UpdateSignPacket, world *level.World, pl *player.Player) {
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

	sendRespawn(connection, p.World)

	pl.SetHP(20)
	sendSetHealth(connection, 20.0)
	sendPlayerPositionAndLook(connection, 0, 0)
	world.MulticastPacket(packets.SpawnPlayerEntityPacket(pl), pl)
	world.MulticastPacket(packets.AlicesRidesBob(pl.GetEntityId(), -1), pl)
	world.MulticastPacket(packets.TeleportPlayerPacket(pl, pl.X, pl.Y, pl.Z, float64(pl.Yaw), float64(pl.Pitch), world), pl)
}

func sendRespawn(connection net.Conn, world byte) {
	respawnPacket := packets.RespawnPacket{
		World: world,
	}
	connection.Write(respawnPacket.Serialize())
}

func sendSetHealth(connection net.Conn, health uint16) {
	setHealthPacket := packets.SetHealthOutPacket{
		Health: health,
	}
	connection.Write(setHealthPacket.Serialize())
}

func outOfBounds(x, z float64) bool {
	return x < worldMinX || x >= worldMaxX || z < worldMinZ || z >= worldMaxZ
}

// rubberBand sends the player back to their last valid position.
func rubberBand(connection net.Conn, pl *player.Player) {
	p := packets.PlayerPositionAndLookOutPacket{
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

func handlePlayerInputInPacket(p packets.PlayerInputInPacket, pl *player.Player, world *level.World) {
	log.Printf("Received PlayerInput packet: Strafe=%.2f Forward=%.2f Jump=%t Sneaking=%t",
		p.StrafeDirection, p.ForwardDirection, p.Jumping, p.Sneaking)
}

// if ok && ridable.ObjectType == constants.ObjectBoat {
//  if p.Y <= ignoreY {
//      if (p.X != pl.X || p.Z != pl.Z) && p.X != 0 && p.Z != 0 {
//          log.Printf("SENTINEL XZ CHANGED: p.X=%.6f p.Z=%.6f | pl.X=%.6f pl.Z=%.6f | dx=%.6f dz=%.6f",
//              p.X, p.Z, pl.X, pl.Z, p.X-pl.X, p.Z-pl.Z)
//      }
//  }
// }

func handlePlayerPositionAndLookInPacket(connection net.Conn, p packets.PlayerPositionAndLookInPacket, pl *player.Player, world *level.World) {
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
			sendSetHealth(connection, 0)
			return
		}
	}
	if y >= 0 {
		pl.BelowZeroHeightCount = 0
	}

	ep := packets.PlayerEntityPositionAndLookPacket(pl, x, y, z, float64(p.Yaw), float64(p.Pitch), world)
	world.MulticastPacket(ep, pl)
	pl.X = x
	pl.Y = y
	pl.Z = z
	pl.Stance = p.Stance
	pl.Yaw = p.Yaw
	pl.Pitch = p.Pitch
	pl.OnGround = p.OnGround
	updateChunks(world, x, z, pl)
}

func handlePlayerPositionInPacket(connection net.Conn, p packets.PlayerPositionInPacket, pl *player.Player, world *level.World) {
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

	// if outOfBounds(x, z) {
	//  rubberBand(connection, pl)
	//  return
	// }

	ep := packets.PlayerEntityPositionPacket(pl, x, y, z, world)
	world.MulticastPacket(ep, pl)
	pl.X = x
	pl.Y = y
	pl.Z = z
	pl.Stance = p.Stance
	pl.OnGround = p.OnGround
	updateChunks(world, x, z, pl)

}

func handlePlayerLookInPacket(p packets.PlayerLookInPacket, pl *player.Player, world *level.World) {
	ep := packets.PlayerEntityLookPacket(pl, float64(p.Yaw), float64(p.Pitch), world)
	world.MulticastPacket(ep, pl)
	pl.Yaw = p.Yaw
	pl.Pitch = p.Pitch
	pl.OnGround = p.OnGround
}

func High8Bits(n uint16) byte {
	return byte(n >> 8)
}

func dropItemFromPlayer(world *level.World, pl *player.Player, typeId int16, metadata uint16, count byte) {
	if count == 0 {
		return
	}

	const dropDistance = 2.0
	yawRad := float64(pl.Yaw) * (math.Pi / 180)
	dirX := -math.Sin(yawRad)
	dirZ := math.Cos(yawRad)
	dropX := pl.X + dirX*dropDistance
	dropZ := pl.Z + dirZ*dropDistance

	spawnDroppedItemPacket := packets.SpawnDroppedItem(
		world,
		typeId,
		count,
		byte(metadata),
		int32(dropX),
		int32(pl.Y),
		int32(dropZ),
		0, 0, 0, 40,
	)
	world.BroadcastPacket(spawnDroppedItemPacket)
}

func handlePlayerDiggingInPacket(connection net.Conn, p packets.PlayerDiggingInPacket, world *level.World, pl *player.Player) {
	if p.Status == 4 {
		dropHeldItemStack(connection, world, pl)
		return
	}
	if pl.IsRiding != -1 {
		return
	}
	world.MulticastPacket(packets.ArmSwing(pl), pl)

	oldBlock := world.GetBlock(p.X, p.Y, p.Z)
	if !shouldProcessDigging(p, pl, oldBlock) {
		return
	}

	damageHeldItemOnDig(pl)

	if oldBlock.TypeId == 0x00 {
		return
	}

	removeMinedBlockEntity(world, p, oldBlock)

	air := level.NewAirBlock()
	//SetBlockAndNotify(world, p.X, int32(p.Y), p.Z, &air)
	world.SetBlockInQueue(p.X, int32(p.Y), p.Z, air)
	world.TriggerFallableUpdate(p.X, int32(p.Y), p.Z, world.SetBlockInQueue)

	blockItem, blockMeta, count := computeMinedDrop(world, p, oldBlock, pl)
	if blockItem == 0 {
		return
	}

	spawnMinedDrop(world, p, blockItem, blockMeta, count)
}

func dropHeldItemStack(connection net.Conn, world *level.World, pl *player.Player) {
	item := pl.Inventory.PeekItem(pl.HotbarSlot)
	if item.TypeId == -1 {
		return
	}
	typeId := item.TypeId
	metadata := item.Metadata
	pl.Inventory.RemoveOne(pl.HotbarSlot)
	sendSetSlot(connection, 0, pl.HotbarSlot, pl.Inventory.Items[pl.HotbarSlot])
	dropItemFromPlayer(world, pl, typeId, metadata, 1)
}

func shouldProcessDigging(p packets.PlayerDiggingInPacket, pl *player.Player, oldBlock level.Block) bool {
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
		oldBlock.TypeId == byte(constants.Sapling.Value)
}

func damageHeldItemOnDig(pl *player.Player) {
	hotBarItem := pl.Inventory.Items[pl.HotbarSlot]
	if !crafting.HasDurability(hotBarItem.TypeId) {
		return
	}
	hotBarItem.Metadata += 1
	if hotBarItem.Metadata == crafting.Durability(hotBarItem.TypeId) {
		pl.Inventory.Items[pl.HotbarSlot] = inventory.EmptyItem()
		sendSetSlot(pl.Connection, 0, pl.HotbarSlot, inventory.EmptyItem())

	} else {
		pl.Inventory.Items[pl.HotbarSlot].Metadata += 1
		sendSetSlot(pl.Connection, 0, pl.HotbarSlot, hotBarItem)
	}
}

func removeMinedBlockEntity(world *level.World, p packets.PlayerDiggingInPacket, oldBlock level.Block) {
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

func computeMinedDrop(world *level.World, p packets.PlayerDiggingInPacket, oldBlock level.Block, pl *player.Player) (blockItem int16, blockMeta byte, count byte) {
	count = 1
	blockItem = int16(oldBlock.TypeId)
	blockMeta = oldBlock.Metadata

	if blockItem == constants.SnowLayer.Value {
		if pl.Inventory.Items[pl.HotbarSlot].IsShovel() {
			log.Printf("Mining Snow with Shovel...")
			blockItem = constants.Snowball.Value
			count = 4
			return blockItem, 0, count
		} else {
			log.Printf("Mining Snow...")
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
		chunk := world.GetLoadedChunk(p.X, p.Z)
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
		chunk := world.GetLoadedChunk(p.X, p.Z)
		delete(chunk.Logic.Growables, bk)

		for i := 1; i <= 3; i++ {
			aboveY := p.Y + byte(i)
			above := world.GetBlock(p.X, aboveY, p.Z)
			if above.TypeId != oldBlock.TypeId {
				break
			}
			air := level.NewAirBlock()
			world.SetBlockInQueue(p.X, int32(aboveY), p.Z, air)
			//SetBlockAndNotify(world, p.X, int32(aboveY), p.Z, &air)
			count++
		}
	}

	if blockItem == constants.Sapling.Value {
		chunk := world.GetLoadedChunk(p.X, p.Z)
		bk := level.BlockKey{X: p.X, Y: p.Y, Z: p.Z}
		delete(chunk.Logic.Growables, bk)
	}

	if blockItem == constants.Grass.Value {
		chunk := world.GetLoadedChunk(p.X, p.Z)
		blockItem = constants.Dirt.Value
		bk := level.BlockKey{X: p.X, Y: p.Y, Z: p.Z}
		delete(chunk.Logic.Growables, bk)
	}

	return blockItem, blockMeta, count
}

func spawnMinedDrop(world *level.World, p packets.PlayerDiggingInPacket, blockItem int16, blockMeta byte, count byte) {
	dropX := int32(p.X)
	dropY := int32(p.Y)
	dropZ := int32(p.Z)
	spawnPacket := packets.SpawnDroppedItem(world, blockItem, count, blockMeta, dropX, dropY, dropZ, 0, 0, 0, 0)
	world.BroadcastPacket(spawnPacket)
	world.TriggerFluidUpdate(dropX, dropY, dropZ, world.SetBlockInQueue)
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

		block := world.GetBlock(int32(bx), byte(by), int32(bz))

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

func handlePlayerBlockPlacementInPacket(connection net.Conn, p packets.PlayerBlockPlacementInPacket, world *level.World, pl *player.Player) {
	oldExisting := world.GetBlock(p.X, byte(p.Y), p.Z)
	logPlacementDebug(pl, oldExisting)

	if openBlockEntityUI(connection, world, pl, p, oldExisting) {
		return
	}

	heldItem := pl.Inventory.PeekItem(pl.HotbarSlot)
	if p.X == -1 && p.Y == 255 && p.Z == -1 && heldItem.TypeId == constants.Boat.Value {
		log.Printf("Player Looks At: x=%f, y=%f, z=%f, yaw=%f, pitch=%f", pl.X, pl.Y, pl.Z, pl.Yaw, pl.Pitch)
		if tryPlaceBoatNoTarget(connection, world, pl) {
			return
		}
		return
	}

	if !canPlaceHeldItem(heldItem) {
		// Only place blocks if block is in hotbar slot
		log.Println("Early return....")
		return
	}

	if heldItem.TypeId == constants.FlintAndSteel.Value {
		handleFlintAndSteelPlacement(world, p, pl, heldItem)
		return
	}

	if tryTillSoil(world, p, oldExisting, heldItem) {
		return
	}

	pl.HotbarLocked.Store(true)
	defer pl.HotbarLocked.Store(false)
	// X/Y/Z are the clicked block; the new block goes on the adjacent face.
	// Face: 0=-Y  1=+Y  2=-Z  3=+Z  4=-X  5=+X
	newX, newY, newZ := placementTargetCoords(p, world)

	// Reject out-of-bounds Y.
	if newY < 0 || newY >= level.CHUNK_SIZE_Y {
		return
	}

	// Reject placement into a chunk that was never sent to the client.
	// cx := level.WorldToChunkCoord(newX)
	// cz := level.WorldToChunkCoord(newZ)
	// if !world.ChunkExists(cx, cz) {
	//  return
	// }

	// Only place into air — don't overwrite existing blocks.
	existing := world.GetBlock(newX, byte(newY), newZ)
	if !existing.IsAir() && !existing.IsLiquid() {
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
	block := level.NewBlockById(p.ItemId, byte(item.Metadata))
	log.Printf("Placing block: TypeId=%d Meta=%d at (%d, %d, %d)", block.TypeId, block.Metadata, newX, newY, newZ)
	if heldItem.TypeId == -1 {
		return
	}

	if block.IsRail() {
		placeRailBlock(world, &block, newX, newY, newZ)
		pl.Inventory.RemoveOne(slot)
		return
	}

	if block.IsDirectional() {
		if !configureDirectionalBlock(world, &block, newX, newY, newZ, heldItem, p) {
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

func logPlacementDebug(pl *player.Player, oldExisting level.Block) {
	if !pl.DebugBlock {
		return
	}
	sendDebugMessage(pl, fmt.Sprintf("Block type=%d meta=%d, light=%d, skylight=%d", oldExisting.TypeId, oldExisting.Metadata, oldExisting.Light, oldExisting.SkyLight))
}

func openBlockEntityUI(connection net.Conn, world *level.World, pl *player.Player, p packets.PlayerBlockPlacementInPacket, oldExisting level.Block) bool {
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
		heldItem.TypeId != constants.FlintAndSteel.Value {
		return false
	}
	return true
}

func handleFlintAndSteelPlacement(world *level.World, p packets.PlayerBlockPlacementInPacket, pl *player.Player, heldItem inventory.Item) {
	fire := level.NewFireBlock()
	world.SetBlockInQueue(p.X, int32(p.Y+1), p.Z, fire)
	//SetBlockAndNotify(world, p.X, int32(p.Y+1), p.Z, &fire)
	if !crafting.HasDurability(heldItem.TypeId) {
		return
	}
	heldItem.Metadata += 1
	if heldItem.Metadata == crafting.Durability(heldItem.TypeId) {
		pl.Inventory.Items[pl.HotbarSlot] = inventory.EmptyItem()
		sendSetSlot(pl.Connection, 0, pl.HotbarSlot, inventory.EmptyItem())

	} else {
		pl.Inventory.Items[pl.HotbarSlot].Metadata += 1
		sendSetSlot(pl.Connection, 0, pl.HotbarSlot, heldItem)
	}
}

func tryTillSoil(world *level.World, p packets.PlayerBlockPlacementInPacket, oldExisting level.Block, heldItem inventory.Item) bool {
	if (oldExisting.TypeId != byte(constants.Dirt.Value) && oldExisting.TypeId != byte(constants.Grass.Value)) || !heldItem.IsHoe() {
		return false
	}
	tilled := level.NewBlockById(constants.Farmland.Value, 0)
	world.SetBlockInQueue(p.X, int32(p.Y), p.Z, tilled)
	//SetBlockAndNotify(world, p.X, int32(p.Y), p.Z, &tilled)
	return true
}

// Face: 0=-Y  1=+Y  2=-Z  3=+Z  4=-X  5=+X
func placementTargetCoords(p packets.PlayerBlockPlacementInPacket, w *level.World) (int32, int, int32) {
	existing := w.GetBlock(p.X, byte(p.Y), p.Z)
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

func tryPlacePlant(connection net.Conn, world *level.World, pl *player.Player, newX int32, newY int, newZ int32, oldExisting level.Block, heldItem inventory.Item) bool {
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
	crop := level.PlantGrowable(world, rule.PlantedBlock, newX, byte(newY), newZ, meta)
	blockChange := packets.BlockChangeOutPacket{
		X:         newX,
		Y:         byte(newY),
		Z:         newZ,
		BlockType: crop.TypeId,
		BlockMeta: crop.Metadata,
	}
	world.BroadcastPacket(blockChange.Serialize())
	pl.Inventory.RemoveOne(pl.HotbarSlot)
	sendSetSlot(connection, 0, pl.HotbarSlot, pl.Inventory.Items[pl.HotbarSlot])
	return true
}

func tryScoopFluidWithBucket(connection net.Conn, world *level.World, pl *player.Player, newX int32, newY int, newZ int32, existing level.Block, heldItem inventory.Item) bool {
	if !existing.IsLiquid() || heldItem.TypeId != constants.Bucket.Value {
		return false
	}
	air := level.NewAirBlock()
	//SetBlockAndNotify(world, newX, int32(newY), newZ, &air)
	world.SetBlockInQueue(newX, int32(newY), newZ, air)
	world.TriggerFluidUpdate(newX, int32(newY), newZ, world.SetBlockInQueue)
	var bucketItem inventory.Item
	if existing.IsWater() {
		bucketItem = inventory.Item{TypeId: constants.WaterBucket.Value, Count: 1}
	} else {
		bucketItem = inventory.Item{TypeId: constants.LavaBucket.Value, Count: 1}
	}
	pl.Inventory.Items[pl.HotbarSlot] = bucketItem
	sendSetSlot(connection, 0, pl.HotbarSlot, bucketItem)
	return true
}

func placeRailBlock(world *level.World, block *level.Block, newX int32, newY int, newZ int32) {
	railIds := map[byte]bool{
		byte(constants.Rail.Value):         true,
		byte(constants.PoweredRail.Value):  true,
		byte(constants.DetectorRail.Value): true,
	}

	hasRail := func(x, y, z int32) bool {
		b := world.GetBlock(x, byte(y), z)
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
	world.SetBlockInQueue(x, y, z, *block)
	//SetBlockAndNotify(world, x, y, z, block)

	// Recalc each flat neighbour now that the new rail exists in the world
	recalcRail := func(nx, ny, nz int32) {
		existing := world.GetBlock(nx, byte(ny), nz)
		if !railIds[existing.TypeId] {
			return
		}
		newMeta := computeMeta(nx, ny, nz)
		if newMeta == existing.Metadata {
			return
		}
		existing.Metadata = newMeta
		world.SetBlockInQueue(nx, ny, nz, existing)
		//SetBlockAndNotify(world, nx, ny, nz, &existing)
	}

	recalcRail(x, y, z-1) // north
	recalcRail(x, y, z+1) // south
	recalcRail(x+1, y, z) // east
	recalcRail(x-1, y, z) // west
}

func configureDirectionalBlock(world *level.World, block *level.Block, newX int32, newY int, newZ int32, heldItem inventory.Item, p packets.PlayerBlockPlacementInPacket) bool {
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
	switch p.Face {
	case 3:
		// West
		block.Metadata = directions.West
	case 2:
		// East
		block.Metadata = directions.East
	case 4:
		// North
		block.Metadata = directions.North
	case 5:
		// South
		block.Metadata = directions.South
	default:
		block.Metadata = 0
	}

	return true
}

func tryPlaceMinecart(connection net.Conn, world *level.World, pl *player.Player, newX int32, newY int, newZ int32, slot int16) {
	beneath := world.GetBlock(newX, byte(newY-1), newZ)
	isRail := beneath.TypeId == byte(constants.Rail.Value) ||
		beneath.TypeId == byte(constants.PoweredRail.Value) ||
		beneath.TypeId == byte(constants.DetectorRail.Value)
	if !isRail {
		return
	}
	entityId := world.NextEntityId()
	spawnPacket := packets.SpawnObject{
		EntityId:      entityId,
		ObjectType:    constants.ObjectMinecart,
		X:             int32(newX * 32),
		Y:             int32(newY * 32),
		Z:             int32(newZ * 32),
		OwnerEntityId: int32(pl.EntityId),
		VelocityX:     0,
		VelocityY:     0,
		VelocityZ:     0,
	}
	world.AddRidable(entityId, pl.GetEntityId(), float64(newX), float64(newY), float64(newZ), 0, 0, 0, 10)
	world.BroadcastPacket(spawnPacket.Serialize())

	pl.Inventory.RemoveOne(slot)
	sendSetSlot(connection, 0, slot, pl.Inventory.Items[slot])
	if pl.Inventory.PeekItem(slot).TypeId == -1 {
		sendEquipmentChangeForHotbarSlot(world, pl)
	}
}

func tryPlaceBoat(connection net.Conn, world *level.World, pl *player.Player, newX int32, newY int, newZ int32, slot int16) {
	entityId := world.NextEntityId()
	// Lift posY by BoatYOffset so the hitbox bottom sits on the block top
	// instead of half-burying the model in the block below.
	spawnY := float64(newY) + entities.BoatYOffset
	spawnPacket := packets.SpawnObject{
		EntityId:      entityId,
		ObjectType:    constants.ObjectBoat,
		X:             int32(math.Floor(float64(newX) * 32)),
		Y:             int32(spawnY * 32),
		Z:             int32(math.Floor(float64(newZ) * 32)),
		OwnerEntityId: int32(pl.EntityId),
		VelocityX:     0,
		VelocityY:     0,
		VelocityZ:     0,
	}
	world.AddRidable(entityId, pl.GetEntityId(), float64(newX), spawnY, float64(newZ), 0, 0, 0, 1)
	world.BroadcastPacket(spawnPacket.Serialize())

	pl.Inventory.RemoveOne(slot)
	sendSetSlot(connection, 0, slot, pl.Inventory.Items[slot])
	if pl.Inventory.PeekItem(slot).TypeId == -1 {
		sendEquipmentChangeForHotbarSlot(world, pl)
	}
}

func tryPlaceFluidFromBucket(connection net.Conn, world *level.World, pl *player.Player, block level.Block, newX int32, newY int, newZ int32, heldItem inventory.Item, slot int16) bool {
	for _, fp := range []fluidPlacement{
		{constants.WaterBucket.Value, func(b *level.Block) bool { return b.IsWater() }, level.NewStillWaterBlock(0)},
		{constants.LavaBucket.Value, func(b *level.Block) bool { return b.IsLava() }, level.NewStillLavaBlock(0)},
	} {
		if !fp.isFluidBlock(&block) && heldItem.TypeId != fp.bucketId {
			continue
		}
		b := fp.newBlock
		//SetBlockAndNotify(world, newX, int32(newY), newZ, &b)
		world.SetBlockInQueue(newX, int32(newY), newZ, b)
		if heldItem.TypeId == fp.bucketId {
			bucket := inventory.Item{TypeId: constants.Bucket.Value, Count: 1, Metadata: 0}
			pl.Inventory.Items[slot] = bucket
			sendSetSlot(connection, 0, slot, bucket)
		} else {
			pl.Inventory.RemoveOne(slot)
			sendSetSlot(connection, 0, slot, pl.Inventory.Items[slot])
		}
		world.TriggerFluidUpdate(newX, int32(newY), newZ, world.SetBlockInQueue)
		return true
	}
	return false
}

func finalizePlacement(connection net.Conn, world *level.World, pl *player.Player, block level.Block, newX int32, newY int, newZ int32, p packets.PlayerBlockPlacementInPacket, slot int16) {
	//SetBlockAndNotify(world, newX, int32(newY), newZ, &block)
	world.SetBlockInQueue(newX, int32(newY), newZ, block)
	world.TriggerFluidUpdate(newX, int32(newY), newZ, world.SetBlockInQueue)
	world.TriggerFallableUpdate(p.X, int32(p.Y), p.Z, world.SetBlockInQueue)

	// Decrement the item in the in-memory inventory and sync to client.
	pl.Inventory.RemoveOne(slot)
	sendSetSlot(connection, 0, slot, pl.Inventory.Items[slot])
	if pl.Inventory.PeekItem(slot).TypeId == -1 {
		sendEquipmentChangeForHotbarSlot(world, pl)
	}

	cx := level.WorldToChunkCoord(int32(newX))
	cz := level.WorldToChunkCoord(int32(newZ))
	coord := level.ChunkCoord{X: cx, Z: cz}
	if !pl.SentChunks.Has(coord.String()) {
		chunk := world.GetOrCreateChunk(cx, cz, level.Empty)
		pre := packets.PreChunkOutPacket{X: cx, Z: cz, Mode: true}
		connection.Write(pre.Serialize())
		mapChunk := packets.MapChunkOutPacket{}
		mapChunk.Apply(*chunk)
		connection.Write(mapChunk.Serialize())
		pl.SentChunks.Set(coord.String(), coord.X, coord.Z)
	}
}

func SetBlockAndNotify(world *level.World, x, y, z int32, block *level.Block) {
	world.SetBlock(x, byte(y), z, *block)
	//log.Printf("SetBlockAndNotify: X=%d Y=%d Z=%d Type=%d Meta=%d", x, y, z, block.TypeId, block.Metadata)
	blockChange := packets.BlockChangeOutPacket{
		X:         x,
		Y:         byte(y),
		Z:         z,
		BlockType: block.TypeId,
		BlockMeta: block.Metadata,
	}
	world.BroadcastPacket(blockChange.Serialize())
}

func handleHoldingChangeInPacket(p packets.HoldingChangeInPacket, pl *player.Player, world *level.World) {
	// Drop the update while a BlockPlacement is in progress to avoid a race
	// where a slot change arriving just after placement resets the wrong slot.
	if pl.HotbarLocked.Load() {
		return
	}
	pl.HotbarSlot = p.Slot + 36
	sendEquipmentChangeForHotbarSlot(world, pl)
}

const pickupRangeSq = 1.5 * 1.5

func CollectNearbyItems(world *level.World) {
	chunks := world.PlayerActiveChunks(1) // 3x3 chunks around each player
	for _, chunk := range chunks {
		logic := chunk.Logic
		for entityId, dropped := range logic.DroppedItems {
			if dropped.PickupDelay > 0 {
				dropped.PickupDelay--
				continue
			}
			itemX := float64(dropped.X)
			itemY := float64(dropped.Y)
			itemZ := float64(dropped.Z)

			for _, pl := range world.Players {
				dx := pl.X - itemX
				dy := pl.Y - itemY
				dz := pl.Z - itemZ
				if dx*dx+dy*dy+dz*dz > pickupRangeSq {
					continue
				}

				slot := pl.Inventory.AddItem(int16(dropped.ItemId), uint16(dropped.Metadata), dropped.Amount)
				if slot < 0 {
					continue
				}
				sendSetSlot(pl.Connection, 0, slot, pl.Inventory.Items[slot])

				collect := packets.CollectItem(entityId, int32(pl.GetEntityId()))
				world.BroadcastPacket(collect)
				world.RemoveDroppedItem(entityId, dropped.X, dropped.Z)
				break
			}
		}
	}
}

func ApplyGravityOnDroppedItems(world *level.World) {
	chunks := world.PlayerActiveChunks(1)
	for _, chunk := range chunks {
		logic := chunk.Logic
		for _, dropped := range logic.DroppedItems {
			below := world.GetBlock(int32(dropped.X), byte(dropped.Y)-1, int32(dropped.Z))
			if below.IsAir() || below.IsLiquid() {
				dropped.Y--
			}
		}
	}
}
