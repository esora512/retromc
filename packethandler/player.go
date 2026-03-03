package packethandler

import (
	"log"
	"net"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func handlePlayerPositionAndLookInPacket(connection net.Conn, p packets.PlayerPositionAndLookInPacket) {
	log.Printf("Player position and look: %+v", p)
}

func handlePlayerPositionInPacket(connection net.Conn, p packets.PlayerPositionInPacket) {
	log.Printf("Player position: %+v", p)
}

// handlePlayerDiggingInPacket handles block-break events.
// Status 2 means the client finished digging — that's when we remove the block and
// credit the item to the player's in-memory inventory.
func handlePlayerDiggingInPacket(connection net.Conn, p packets.PlayerDiggingInPacket, world *level.World, pl *player.Player) {
	if p.Status != 2 {
		return
	}

	oldBlock := world.GetBlock(p.X, p.Y, p.Z)
	// Don't credit air — player somehow finished digging an empty cell.
	if oldBlock.TypeId == 0x00 {
		return
	}

	air := level.NewAirBlock()
	world.SetBlock(p.X, p.Y, p.Z, air)

	// Notify client of the block change.
	blockChange := packets.BlockChangeOutPacket{
		X:         p.X,
		Y:         p.Y,
		Z:         p.Z,
		BlockType: air.TypeId,
		BlockMeta: air.Metadata,
	}
	connection.Write(blockChange.Serialize())

	// Add the mined block to the in-memory inventory.
	// AddItem handles: stack-on-existing, first-empty-slot, and full-inventory cases.
	slot := pl.Inventory.AddItem(int16(oldBlock.TypeId))
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
	if existing.TypeId != 0x00 {
		return
	}

	// Verify the player actually has the item they're trying to place.
	//slot := pl.Inventory.FindFirstSlotWith(p.ItemId)
	slot := pl.HotbarSlot
	block := level.NewBlockById(p.ItemId)
	world.SetBlock(newX, byte(newY), newZ, block)

	// Notify client of the block change.
	blockChange := packets.BlockChangeOutPacket{
		X:         newX,
		Y:         byte(newY),
		Z:         newZ,
		BlockType: block.TypeId,
		BlockMeta: block.Metadata,
	}
	connection.Write(blockChange.Serialize())

	// Decrement the item in the in-memory inventory and sync to client.
	pl.Inventory.RemoveOneFromSlot(slot)
	sendSetSlot(connection, 0, slot, pl.Inventory.Items[slot])
}
