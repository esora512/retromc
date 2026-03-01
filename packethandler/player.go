package packethandler

import (
	"net"

	"log"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
)

func handlePlayerPositionAndLookInPacket(connection net.Conn, p packets.PlayerPositionAndLookInPacket) {
	log.Printf("Player position and look: %+v", p)
}

func handlePlayerPositionInPacket(connection net.Conn, p packets.PlayerPositionInPacket) {
	log.Printf("Player position: %+v", p)
}

// handlePlayerDiggingInPacket handles block-break events.
// Status 2 means the client finished digging — that's when we actually remove the block.
func handlePlayerDiggingInPacket(connection net.Conn, p packets.PlayerDiggingInPacket, world *level.World) {
	if p.Status != 2 {
		return
	}

	oldBlock := world.GetBlock(p.X, p.Y, p.Z)
	log.Printf("Mined block: %+v", oldBlock)
	air := level.NewAirBlock()
	world.SetBlock(p.X, p.Y, p.Z, air)

	blockChange := packets.BlockChangeOutPacket{
		X:         p.X,
		Y:         p.Y,
		Z:         p.Z,
		BlockType: air.TypeId,
		BlockMeta: air.Metadata,
	}
	connection.Write(blockChange.Serialize())

	//TODO: Refine this; check for empty slots; check for existing same block and increase count
	setItemInInventory(connection, int16(oldBlock.TypeId), 1, 9)
}

func handlePlayerBlockPlacementInPacket(connection net.Conn, p packets.PlayerBlockPlacementInPacket, world *level.World) {
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
	log.Printf("Place: %+v", p)

	// Reject out-of-bounds Y (e.g. face offset pushed below 0 or above 127).
	if newY < 0 || newY >= level.CHUNK_SIZE_Y {
		return
	}

	// Reject placement into a chunk that was never sent to the client.
	cx := level.WorldToChunkCoord(newX)
	cz := level.WorldToChunkCoord(newZ)
	if !world.ChunkExists(cx, cz) {
		return
	}

	// // Only place into air — don't overwrite existing blocks.
	existing := world.GetBlock(newX, byte(newY), newZ)
	if existing.TypeId != 0x00 {
		return
	}

	block := level.NewBlockById(p.ItemId)
	world.SetBlock(newX, byte(newY), newZ, block)

	blockChange := packets.BlockChangeOutPacket{
		X:         newX,
		Y:         byte(newY),
		Z:         newZ,
		BlockType: block.TypeId,
		BlockMeta: block.Metadata,
	}
	connection.Write(blockChange.Serialize())
}
