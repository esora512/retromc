package packethandler

import (
	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

var claimable = map[byte]bool{
	byte(constants.Stone.Value):      true,
	byte(constants.SignGround.Value): true,
	byte(constants.SignWall.Value):   true,
}

func sendBlockToPlayer(world *level.World, pl *player.Player, x int32, y byte, z int32, b constants.WBlock) {
	p := packets.SetBlockPacket{X: x, Y: y, Z: z, BlockType: b.TypeId, BlockMeta: b.Metadata}
	pl.Connection.Write(p.Serialize())
	if lines, ok := world.SignLines(x, int32(y), z, pl.Dimension); ok {
		sign := packets.UpdateSignPacket{X: x, Y: int16(y), Z: z, Text1: lines[0], Text2: lines[1], Text3: lines[2], Text4: lines[3]}
		pl.Connection.Write(sign.Serialize())
	}
}

func handleClaimCommand(pl *player.Player) {
	pl.ClaimMode = !pl.ClaimMode
	if pl.ClaimMode {
		sendDebugMessage(pl, "Claim mode enabled, r-click blocks w/ hand to un/claim")
	} else {
		sendDebugMessage(pl, "Claim mode disabled")
	}
}

func toggleClaim(world *level.World, pl *player.Player, x, y, z int32) {
	if owner, ok := world.ClaimOwner(x, y, z, pl.Dimension); ok {
		world.Unclaim(x, y, z, pl.Dimension)
		sendDebugMessage(pl, "Unclaimed block owned by "+owner)
	} else if claimable[world.GetBlock(x, byte(y), z, pl.Dimension).TypeId] {
		world.Claim(pl.Username, x, y, z, pl.Dimension)
		sendDebugMessage(pl, "Claimed block")
	} else {
		sendDebugMessage(pl, "§cOnly stone and signs can be claimed")
	}
}
