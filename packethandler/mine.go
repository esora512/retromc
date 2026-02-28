package packethandler

import (
	"net"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
)

// handleMineInPacket handles block-break events.
// Status 2 means the client finished digging — that's when we actually remove the block.
func handleMineInPacket(connection net.Conn, p packets.MineInPacket, world *level.World) {
	if p.Status != 2 {
		return
	}

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
}

