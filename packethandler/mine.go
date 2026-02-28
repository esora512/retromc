package packethandler

import (
	"net"

	"log"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

// handleMineInPacket handles block-break events.
// Status 2 means the client finished digging — that's when we actually remove the block.
func handleMineInPacket(connection net.Conn, p packets.MineInPacket, world *level.World) {
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
	setSlot := packets.SetSlotOutPacket{
		WindowId: 0,
		Slot:     9,
		Item:     player.Item{TypeId: int16(oldBlock.TypeId), Count: 1, Uses: 0},
	}
	connection.Write(setSlot.Serialize())
}
