package packethandler

import (
	"fmt"
	"strconv"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/player"
)

const fillMaxBlocks = 1 << 18

func handleFillCommand(pl *player.Player, world *level.World, args []string) {
	switch len(args) {
	case 0:
		pl.FillSelect = !pl.FillSelect
		if pl.FillSelect {
			pl.FillCorners = nil
			sendDebugMessage(pl, "Fill select enabled, r-click two blocks w/ hand")
		} else {
			sendDebugMessage(pl, "Fill select disabled")
		}
	case 1:
		if len(pl.FillCorners) != 2 {
			sendDebugMessage(pl, "Select two corners first with /fill")
			return
		}
		if pl.FillDim != pl.Dimension {
			sendDebugMessage(pl, "Selected corners are in another dimension")
			return
		}
		fill(pl, world, args[0], pl.FillCorners[0], pl.FillCorners[1])
	case 7:
		var c [6]int32
		for i, a := range args[1:] {
			v, err := strconv.ParseInt(a, 10, 32)
			if err != nil {
				sendUsage(pl, "/fill")
				return
			}
			c[i] = int32(v)
		}
		fill(pl, world, args[0], [3]int32{c[0], c[1], c[2]}, [3]int32{c[3], c[4], c[5]})
	default:
		sendUsage(pl, "/fill")
	}
}

func selectFillCorner(pl *player.Player, x, y, z int32) {
	if len(pl.FillCorners) == 2 || pl.FillDim != pl.Dimension {
		pl.FillCorners = nil
	}
	pl.FillDim = pl.Dimension
	pl.FillCorners = append(pl.FillCorners, [3]int32{x, y, z})
	sendDebugMessage(pl, fmt.Sprintf("Corner %d set to x=%d, y=%d, z=%d", len(pl.FillCorners), x, y, z))
}

func fill(pl *player.Player, world *level.World, blockName string, c1, c2 [3]int32) {
	b := constants.GetBlockByName(blockName)
	if b.Value == -1 {
		sendDebugMessage(pl, fmt.Sprintf("Unknown block: %s", blockName))
		return
	}
	block := constants.NewBlockById(b.Value, byte(b.Meta))

	minX, maxX := min(c1[0], c2[0]), max(c1[0], c2[0])
	minY, maxY := max(min(c1[1], c2[1]), 0), min(max(c1[1], c2[1]), level.CHUNK_SIZE_Y-1)
	minZ, maxZ := min(c1[2], c2[2]), max(c1[2], c2[2])
	if minY > maxY {
		sendDebugMessage(pl, "Selection is outside the world height")
		return
	}
	size := int64(maxX-minX+1) * int64(maxY-minY+1) * int64(maxZ-minZ+1)
	if size > fillMaxBlocks {
		sendDebugMessage(pl, fmt.Sprintf("Selection too large: %d blocks (max %d)", size, fillMaxBlocks))
		return
	}

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			for z := minZ; z <= maxZ; z++ {
				world.BreakContainer(x, y, z, world.GetBlock(x, byte(y), z, pl.Dimension).TypeId, pl.Dimension)
				world.SetBlockInQueue(x, y, z, block, pl.Dimension)
			}
		}
	}
	sendDebugMessage(pl, fmt.Sprintf("Filled %d %s block(s)", size, blockName))
}
