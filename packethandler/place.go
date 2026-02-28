package packethandler

import (
	"net"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
)

func handlePlaceInPacket(connection net.Conn, p packets.PlaceInPacket, world *level.World) {
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
