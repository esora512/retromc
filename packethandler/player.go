package packethandler

import (
	"log"
	"net"

	"github.com/leNicDev/retromc/constants"
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

func handlePlayerPositionAndLookInPacket(connection net.Conn, p packets.PlayerPositionAndLookInPacket, pl *player.Player, world *level.World) {
	x, y, z := p.X, p.Y, p.Z
	// If p.Y is set -999 by Client, that means the Server should ignore positions
	// Happens if the player is entering/leaving minecarts
	// For now there is no elegant way to make this work via flags due to slight delays and position mismatches
	// TODO: It might be possible to make it work via channels; but again, not sure if it's worth it...
	if p.Y <= ignoreY && pl.IsRiding != -1 {
		maybeRidable := world.Entities[pl.IsRiding]
		ridable, _ := maybeRidable.(*level.RideableEntity)
		x, y, z = ridable.X, ridable.Y, ridable.Z
		if p.Yaw == pl.Yaw && p.Pitch == pl.Pitch {
			// Only multicast if different from before
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
	x, y, z := p.X, p.Y, p.Z
	// If p.Y is set -999 by Client, that means the Server should ignore positions
	// Happens if the player is entering/leaving minecarts
	// For now there is no elegant way to make this work via flags due to slight delays and position mismatches
	if p.Y <= ignoreY {
		return
	}
	if outOfBounds(x, z) {
		rubberBand(connection, pl)
		return
	}
	ep := packets.PlayerEntityPositionPacket(pl, x, y, z, world)
	world.MulticastPacket(ep, pl)

	// TODO: Unclear if we need to do this for precision & avoiding drift or overkill
	// pl.X = float64(int32(math.Floor(x * 32))) / 32.0
	// pl.Y = float64(int32(math.Floor(y * 32))) / 32.0
	// pl.Z = float64(int32(math.Floor(z * 32))) / 32.0
	pl.X = x
	pl.Y = y
	pl.Z = z
	pl.Stance = p.Stance
	pl.OnGround = p.OnGround
}

func handlePlayerLookInPacket(p packets.PlayerLookInPacket, pl *player.Player, world *level.World) {
	ep := packets.PlayerEntityLookPacket(pl, float64(p.Yaw), float64(p.Pitch), world)
	log.Printf("Multicasting player look yaw=%.2f pitch=%.2f", p.Yaw, p.Pitch)
	world.MulticastPacket(ep, pl)
	pl.Yaw = p.Yaw
	pl.Pitch = p.Pitch
	pl.OnGround = p.OnGround
}

// handlePlayerDiggingInPacket handles block-break events.
// Status 2 means the client finished digging — that's when we remove the block and
// credit the item to the player's in-memory inventory.
func handlePlayerDiggingInPacket(connection net.Conn, p packets.PlayerDiggingInPacket, world *level.World, pl *player.Player) {
	log.Printf("Face %d Status %d", p.Face, p.Status)
	if pl.IsRiding != -1 {
		return
	}
	world.MulticastPacket(packets.ArmSwing(pl), pl)

	if p.Status != 2 {
		return
	}
	oldBlock := world.GetBlock(p.X, p.Y, p.Z)
	if oldBlock.TypeId == 0x00 {
		return
	}

	if oldBlock.TypeId == byte(constants.Chest.Value) {
		inventory.RemoveChest(p.X, int32(p.Y), p.Z)
	}

	if oldBlock.TypeId == byte(constants.Dispenser.Value) {
		inventory.RemoveDispenser(p.X, int32(p.Y), p.Z)
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
	log.Printf("PlayerBlockPlacement: %+v", p)
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
	if heldItem.TypeId > 96 && heldItem.TypeId != constants.Minecart.Value && heldItem.TypeId != constants.Sign.Value {
		// Only place blocks if block is in hotbar slot
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
	if !existing.IsAir() {
		return
	}

	// Verify the player actually has the item they're trying to place.
	//slot := pl.Inventory.FindFirstSlotWith(p.ItemId)
	slot := pl.HotbarSlot
	item := pl.Inventory.PeekItem(slot)
	block := level.NewBlockById(p.ItemId, item.Metadata)

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

		log.Println("Face", p.Face)
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
			ObjectType:    10, // 10 for minecart
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

	world.SetBlock(newX, byte(newY), newZ, block)

	// Notify all players of the block change.
	blockChange := packets.BlockChangeOutPacket{
		X:         newX,
		Y:         byte(newY),
		Z:         newZ,
		BlockType: block.TypeId,
		BlockMeta: block.Metadata,
	}
	world.BroadcastPacket(blockChange.Serialize())

	// Decrement the item in the in-memory inventory and sync to client.
	pl.Inventory.RemoveOne(slot)
	sendSetSlot(connection, 0, slot, pl.Inventory.Items[slot])
	if pl.Inventory.PeekItem(slot).TypeId == -1 {
		sendEquipmentChangeForHotbarSlot(world, pl)
	}
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
