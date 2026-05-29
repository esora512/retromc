package packethandler

import (
	"log"
	"math"
	"net"

	"github.com/leNicDev/retromc/constants"
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
	sendSetHealth(connection, 20.0)
	sendPlayerPositionAndLook(connection)

	world.MulticastPacket(packets.AlicesRidesBob(pl.GetEntityId(), -1), pl)
	world.MulticastPacket(packets.TeleportPlayerPacket(pl, pl.X, pl.Y, pl.Z, float64(pl.Yaw), float64(pl.Pitch), world), pl)
}

func sendRespawn(connection net.Conn, world byte) {
	respawnPacket := packets.RespawnPacket{
		World: world,
	}
	connection.Write(respawnPacket.Serialize())
}

func sendSetHealth(connection net.Conn, health float32) {
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

func logMovementDirection(pl *player.Player, newX, newZ float64) {
	dx := newX - pl.X
	dz := newZ - pl.Z

	if math.Abs(dx) < 0.01 && math.Abs(dz) < 0.01 {
		return // no meaningful movement, skip
	}

	var dirX, dirZ string
	if dx > 0.01 {
		dirX = "E"
	} else if dx < -0.01 {
		dirX = "W"
	}
	if dz > 0.01 {
		dirZ = "S"
	} else if dz < -0.01 {
		dirZ = "N"
	}

	dir := dirX + dirZ
	if dir == "" {
		dir = "?"
	}
	log.Printf("[move] %s  dx=%.3f dz=%.3f", dir, dx, dz)
}

func handlePlayerInputInPacket(p packets.PlayerInputInPacket, pl *player.Player, world *level.World) {
	log.Printf("Received PlayerInput packet: Strafe=%.2f Forward=%.2f Jump=%t Sneaking=%t",
		p.StrafeDirection, p.ForwardDirection, p.Jumping, p.Sneaking)
}

// if ok && ridable.ObjectType == constants.ObjectBoat {
// 	if p.Y <= ignoreY {
// 		if (p.X != pl.X || p.Z != pl.Z) && p.X != 0 && p.Z != 0 {
// 			log.Printf("SENTINEL XZ CHANGED: p.X=%.6f p.Z=%.6f | pl.X=%.6f pl.Z=%.6f | dx=%.6f dz=%.6f",
// 				p.X, p.Z, pl.X, pl.Z, p.X-pl.X, p.Z-pl.Z)
// 		}
// 	}
// }

func handlePlayerPositionAndLookInPacket(connection net.Conn, p packets.PlayerPositionAndLookInPacket, pl *player.Player, world *level.World) {
	if pl.IsRiding != -1 {
		maybeRidable := world.Entities[pl.IsRiding]
		ridable, ok := maybeRidable.(*entities.RideableEntity)
		if ok && ridable.ObjectType == constants.ObjectBoat {
			if p.Y <= ignoreY && ((p.X != pl.X || p.Z != pl.Z) && (p.X != 0 && p.Z != 0)) {
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
		if p.Yaw == pl.Yaw && p.Pitch == pl.Pitch {
			return
		}
	}

	if outOfBounds(x, z) {
		rubberBand(connection, pl)
		return
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
}

func handlePlayerPositionInPacket(connection net.Conn, p packets.PlayerPositionInPacket, pl *player.Player, world *level.World) {
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

	if outOfBounds(x, z) {
		rubberBand(connection, pl)
		return
	}

	ep := packets.PlayerEntityPositionPacket(pl, x, y, z, world)
	world.MulticastPacket(ep, pl)
	pl.X = x
	pl.Y = y
	pl.Z = z
	pl.Stance = p.Stance
	pl.OnGround = p.OnGround
}

func handlePlayerLookInPacket(p packets.PlayerLookInPacket, pl *player.Player, world *level.World) {
	ep := packets.PlayerEntityLookPacket(pl, float64(p.Yaw), float64(p.Pitch), world)
	world.MulticastPacket(ep, pl)
	pl.Yaw = p.Yaw
	pl.Pitch = p.Pitch
	pl.OnGround = p.OnGround
}

// handlePlayerDiggingInPacket handles block-break events.
// Status 2 means the client finished digging — that's when we remove the block and
// credit the item to the player's in-memory inventory.
func handlePlayerDiggingInPacket(connection net.Conn, p packets.PlayerDiggingInPacket, world *level.World, pl *player.Player) {
	if pl.IsRiding != -1 {
		return
	}
	world.MulticastPacket(packets.ArmSwing(pl), pl)

	oldBlock := world.GetBlock(p.X, p.Y, p.Z)
	finishedDigging := p.Status == 2 || (pl.IsCreative && p.Status == 0)
	if !finishedDigging && oldBlock.TypeId != byte(constants.Wheat.Value) {
		return
	}

	if oldBlock.TypeId == 0x00 {
		return
	}

	if oldBlock.TypeId == byte(constants.Chest.Value) {
		inventory.RemoveChest(p.X, int32(p.Y), p.Z)
	}

	if oldBlock.TypeId == byte(constants.Dispenser.Value) {
		inventory.RemoveDispenser(p.X, int32(p.Y), p.Z)
	}

	if oldBlock.TypeId == byte(constants.Furnace.Value) {
		inventory.RemoveFurnace(p.X, int32(p.Y), p.Z)
	}

	air := level.NewAirBlock()
	world.SetBlock(p.X, p.Y, p.Z, air)

	// Notify all players of the block change.
	blockChange := packets.BlockChangeOutPacket{
		X:         p.X,
		Y:         p.Y,
		Z:         p.Z,
		BlockType: air.TypeId,
		BlockMeta: air.Metadata,
	}
	world.BroadcastPacket(blockChange.Serialize())

	// Add the mined block to the in-memory inventory.
	// AddItem handles: stack-on-existing, first-empty-slot, and full-inventory cases.
	blockItem := int16(oldBlock.TypeId)
	blockMeta := oldBlock.Metadata
	if blockItem == constants.Stone.Value {
		blockItem = constants.Cobblestone.Value
	}

	if blockItem == constants.SignGround.Value {
		blockItem = constants.Sign.Value
	}

	if blockItem == constants.Rail.Value || blockItem == constants.PoweredRail.Value || blockItem == constants.DetectorRail.Value {
		blockMeta = byte(0)
	}

	if blockItem == constants.Wheat.Value && blockMeta == 0 {
		blockItem = constants.Seeds.Value
	}

	slot := pl.Inventory.AddItem(blockItem, blockMeta, 1)
	if slot < 0 {
		return
	}
	// Tell the client about the updated slot.
	sendSetSlot(connection, 0, slot, pl.Inventory.Items[slot])
}

// handlePlayerBlockPlacementInPacket handles block-place events.
// It decrements the placed item from the player's in-memory inventory.
// HotbarSlot is locked for the duration so that a HoldingChange packet
// arriving concurrently cannot overwrite it mid-placement.
func handlePlayerBlockPlacementInPacket(connection net.Conn, p packets.PlayerBlockPlacementInPacket, world *level.World, pl *player.Player) {
	oldExisting := world.GetBlock(p.X, byte(p.Y), p.Z)
	if oldExisting.TypeId == byte(constants.CraftingTable.Value) {
		p := packets.NewCraftingTable()
		connection.Write(p.Serialize())
		pl.InventoryType = player.WorkbenchInventory
		return
	}

	if oldExisting.TypeId == byte(constants.Chest.Value) {
		chest := inventory.GetChest(p.X, int32(p.Y), p.Z)
		chestPacket := packets.NewChest(byte(chest.Size))
		connection.Write(chestPacket.Serialize())
		pl.InventoryType = player.ChestInventory
		pl.Chest.X = int32(p.X)
		pl.Chest.Y = int32(p.Y)
		pl.Chest.Z = int32(p.Z)
		sendChestContents(connection, chest)
		return
	}

	if oldExisting.TypeId == byte(constants.Dispenser.Value) {
		dispenser := inventory.GetDispenser(p.X, int32(p.Y), p.Z)
		if dispenser == nil {
			return
		}
		dispenserPacket := packets.NewDispenser()
		connection.Write(dispenserPacket.Serialize())
		pl.InventoryType = player.DispenserInventory
		pl.Dispenser.X = int32(p.X)
		pl.Dispenser.Y = int32(p.Y)
		pl.Dispenser.Z = int32(p.Z)
		sendDispenserContents(connection, dispenser)
		return
	}

	if oldExisting.TypeId == byte(constants.Furnace.Value) {
		furnace := inventory.GetFurnace(p.X, int32(p.Y), p.Z)
		if furnace == nil {
			return
		}
		furnacePacket := packets.NewFurnace()
		connection.Write(furnacePacket.Serialize())
		pl.InventoryType = player.FurnaceInventory
		pl.Furnace.X = int32(p.X)
		pl.Furnace.Y = int32(p.Y)
		pl.Furnace.Z = int32(p.Z)
		sendFurnaceContents(connection, furnace)
		return
	}

	heldItem := pl.Inventory.PeekItem(pl.HotbarSlot)
	if heldItem.TypeId > 96 &&
		heldItem.TypeId != constants.Boat.Value &&
		heldItem.TypeId != constants.Minecart.Value &&
		heldItem.TypeId != constants.Sign.Value &&
		heldItem.TypeId != constants.WaterBucket.Value &&
		heldItem.TypeId != constants.Bucket.Value &&
		heldItem.TypeId != constants.LavaBucket.Value &&
		!heldItem.IsHoe() &&
		heldItem.TypeId != constants.Seeds.Value {
		// Only place blocks if block is in hotbar slot
		return
	}

	if oldExisting.TypeId == byte(constants.Dirt.Value) && heldItem.IsHoe() {
		tilled := level.NewBlockById(constants.Farmland.Value, 0)
		world.SetBlock(p.X, byte(p.Y), p.Z, tilled)
		blockChange := packets.BlockChangeOutPacket{
			X:         p.X,
			Y:         byte(p.Y),
			Z:         p.Z,
			BlockType: tilled.TypeId,
			BlockMeta: tilled.Metadata,
		}
		world.BroadcastPacket(blockChange.Serialize())
		return
	}

	pl.HotbarLocked.Store(true)
	defer pl.HotbarLocked.Store(false)
	// X/Y/Z are the clicked block; the new block goes on the adjacent face.
	// Face: 0=-Y  1=+Y  2=-Z  3=+Z  4=-X  5=+X
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

	// Reject out-of-bounds Y.
	if newY < 0 || newY >= level.CHUNK_SIZE_Y {
		return
	}

	// Reject placement into a chunk that was never sent to the client.
	cx := level.WorldToChunkCoord(newX)
	cz := level.WorldToChunkCoord(newZ)
	if !world.ChunkExists(cx, cz) {
		return
	}

	// Only place into air — don't overwrite existing blocks.
	existing := world.GetBlock(newX, byte(newY), newZ)
	if !existing.IsAir() && !existing.IsLiquid() {
		return
	}

	if oldExisting.TypeId == byte(constants.Farmland.Value) && heldItem.TypeId == constants.Seeds.Value {
		crop := level.NewBlockById(constants.Wheat.Value, 0)
		world.SetBlock(newX, byte(newY), newZ, crop)
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
		return
	}

	if existing.IsLiquid() && heldItem.TypeId == constants.Bucket.Value {
		air := level.NewAirBlock()
		SetBlockAndNotify(world, newX, int32(newY), newZ, &air)
		delete(world.WaterSources, level.BlockKey{X: newX, Y: byte(newY), Z: newZ})
		delete(world.FlowingWater, level.BlockKey{X: newX, Y: byte(newY), Z: newZ})
		delete(world.LavaSources, level.BlockKey{X: newX, Y: byte(newY), Z: newZ})
		delete(world.FlowingLava, level.BlockKey{X: newX, Y: byte(newY), Z: newZ})

		var bucketItem inventory.Item
		if existing.IsWater() {
			bucketItem = inventory.Item{TypeId: constants.WaterBucket.Value, Count: 1}
		} else {
			bucketItem = inventory.Item{TypeId: constants.LavaBucket.Value, Count: 1}
		}
		pl.Inventory.Items[pl.HotbarSlot] = bucketItem
		sendSetSlot(connection, 0, pl.HotbarSlot, bucketItem)
		return
	}

	// Verify the player actually has the item they're trying to place.
	//slot := pl.Inventory.FindFirstSlotWith(p.ItemId)
	slot := pl.HotbarSlot
	item := pl.Inventory.PeekItem(slot)
	block := level.NewBlockById(p.ItemId, item.Metadata)
	if heldItem.TypeId == -1 {
		return
	}

	if block.IsRail() {
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

		broadcastBlock := func(x, y, z int32, typeId, meta byte) {
			pkt := packets.BlockChangeOutPacket{
				X:         x,
				Y:         byte(y),
				Z:         z,
				BlockType: typeId,
				BlockMeta: meta,
			}
			world.BroadcastPacket(pkt.Serialize())
		}

		x, y, z := int32(newX), int32(newY), int32(newZ)

		// Place the new rail with computed metadata
		block.Metadata = computeMeta(x, y, z)
		world.SetBlock(x, byte(y), z, block)
		broadcastBlock(x, y, z, block.TypeId, block.Metadata)

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
			world.SetBlock(nx, byte(ny), nz, existing)
			broadcastBlock(nx, ny, nz, existing.TypeId, newMeta)
		}

		recalcRail(x, y, z-1) // north
		recalcRail(x, y, z+1) // south
		recalcRail(x+1, y, z) // east
		recalcRail(x-1, y, z) // west
		pl.Inventory.RemoveOne(slot)
		return
	}

	if block.IsDirectional() {
		if block.TypeId == byte(constants.Chest.Value) {
			check := inventory.PlaceChest(int32(newX), int32(newY), int32(newZ))
			if !check {
				return
			}
		}

		if block.TypeId == byte(constants.Dispenser.Value) {
			check := inventory.PlaceDispenser(int32(newX), int32(newY), int32(newZ))
			if !check {
				return
			}
		}

		if block.TypeId == byte(constants.Furnace.Value) {
			check := inventory.PlaceFurnace(int32(newX), int32(newY), int32(newZ))
			if !check {
				return
			}
		}

		if block.TypeId == byte(constants.Sign.Value) {
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
	}

	// Handle minecart placement
	if heldItem.TypeId == constants.Minecart.Value {
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
		return
	}

	if heldItem.TypeId == constants.Boat.Value {
		// beneath := world.GetBlock(newX, byte(newY-1), newZ)
		// isWater := beneath.IsWater()
		// if !isWater {
		// 	return
		// }
		entityId := world.NextEntityId()
		// Lift posY by BoatYOffset so the hitbox bottom sits on the block top
		// instead of half-burying the model in the block below.
		spawnY := float64(newY) + entities.BoatYOffset
		spawnPacket := packets.SpawnObject{
			EntityId:      entityId,
			ObjectType:    constants.ObjectBoat,
			X:             int32(newX * 32),
			Y:             int32(spawnY * 32),
			Z:             int32(newZ * 32),
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
		return
	}

	for _, fp := range []fluidPlacement{
		{constants.WaterBucket.Value, func(b *level.Block) bool { return b.IsWater() }, level.NewStillWaterBlock(0)},
		{constants.LavaBucket.Value, func(b *level.Block) bool { return b.IsLava() }, level.NewStillLavaBlock(0)},
	} {
		if !fp.isFluidBlock(&block) && heldItem.TypeId != fp.bucketId {
			continue
		}
		b := fp.newBlock
		SetBlockAndNotify(world, newX, int32(newY), newZ, &b)
		if heldItem.TypeId == fp.bucketId {
			bucket := inventory.Item{TypeId: constants.Bucket.Value, Count: 1, Metadata: 0}
			pl.Inventory.Items[slot] = bucket
			sendSetSlot(connection, 0, slot, bucket)
		} else {
			pl.Inventory.RemoveOne(slot)
			sendSetSlot(connection, 0, slot, pl.Inventory.Items[slot])
		}
		return
	}
	SetBlockAndNotify(world, newX, int32(newY), newZ, &block)

	// Decrement the item in the in-memory inventory and sync to client.
	pl.Inventory.RemoveOne(slot)
	sendSetSlot(connection, 0, slot, pl.Inventory.Items[slot])
	if pl.Inventory.PeekItem(slot).TypeId == -1 {
		sendEquipmentChangeForHotbarSlot(world, pl)
	}
}

func SetBlockAndNotify(world *level.World, x, y, z int32, block *level.Block) {
	world.SetBlock(x, byte(y), z, *block)
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
