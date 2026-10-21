package packethandler

import (
	"fmt"
	"log"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func sendBlockToPlayer(world *level.World, pl *player.Player, x int32, y byte, z int32, b constants.WBlock) {
	p := packets.SetBlockPacket{X: x, Y: y, Z: z, BlockType: b.TypeId, BlockMeta: b.Metadata}
	pl.Connection.Write(p.Serialize())
	if lines, ok := world.SignLines(x, int32(y), z, pl.Dimension); ok {
		sign := packets.UpdateSignPacket{X: x, Y: int16(y), Z: z, Text1: lines[0], Text2: lines[1], Text3: lines[2], Text4: lines[3]}
		pl.Connection.Write(sign.Serialize())
	}
}

func handleClaimCommand(pl *player.Player, world *level.World, args []string) {
	if len(args) == 0 {
		toggleSelect(pl)
		return
	}
	if args[0] != "add" && args[0] != "remove" {
		sendUsage(pl, "/claim")
		return
	}
	c1, c2, ok := selection(pl, "/claim", args[1:])
	if !ok {
		return
	}
	box := level.ClaimBox{
		Min: [3]int32{min(c1[0], c2[0]), min(c1[1], c2[1]), min(c1[2], c2[2])},
		Max: [3]int32{max(c1[0], c2[0]), max(c1[1], c2[1]), max(c1[2], c2[2])},
		Dim: pl.Dimension,
	}
	area := fmt.Sprintf("(%d,%d,%d) to (%d,%d,%d) dim %d", box.Min[0], box.Min[1], box.Min[2], box.Max[0], box.Max[1], box.Max[2], box.Dim)
	if args[0] == "add" {
		world.Claim(pl.Username, box)
		log.Printf("CLAIM: %s claimed %s", pl.Username, area)
		sendDebugMessage(pl, "Claimed "+area)
		return
	}
	n := world.Unclaim(box)
	log.Printf("CLAIM: %s removed %d claim(s) touching %s", pl.Username, n, area)
	sendDebugMessage(pl, fmt.Sprintf("Removed %d claim(s)", n))
}
