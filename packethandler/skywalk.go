package packethandler

import (
	"math"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

const skywalkRadius = 2

func updateSkywalk(world *level.World, pl *player.Player) {
	want := map[[4]int32]bool{}
	climbed := false
	prev := pl.SkywalkPrev
	pl.SkywalkPrev = [3]float64{pl.X, pl.Y, pl.Z}
	if pl.Skywalk && pl.HP > 0 {
		feet := int32(math.Floor(pl.Y)) - 1
		jumpedInPlace := !pl.OnGround && feet == pl.SkywalkY && pl.Y > prev[1] && pl.Y > float64(pl.SkywalkY+1)+0.1 &&
			math.Abs(pl.X-prev[0]) < 0.01 && math.Abs(pl.Z-prev[2]) < 0.01
		switch {
		// teleport, dimension change or big fall: restart the platform under the player
		case pl.SkywalkDim != pl.Dimension || feet < pl.SkywalkY-3 || feet > pl.SkywalkY+3:
			pl.SkywalkY, pl.SkywalkDim = feet, pl.Dimension
			pl.SkywalkWaitForLanding = false
		// sneaking while standing on the platform: lower it one block
		case pl.IsSneaking && pl.OnGround && feet == pl.SkywalkY:
			pl.SkywalkY--
			pl.SkywalkWaitForLanding = true
		// platform just moved: hold it until the player stands on the new layer, ignoring stale positions
		case pl.SkywalkWaitForLanding:
			pl.SkywalkWaitForLanding = !(pl.OnGround && feet == pl.SkywalkY)
		// jumped without moving sideways: raise it one block and put the player on top
		case jumpedInPlace:
			pl.SkywalkY++
			pl.SkywalkWaitForLanding = true
			climbed = true
		// standing on something higher: move the platform up to that level
		case pl.OnGround && feet > pl.SkywalkY:
			pl.SkywalkY = feet
		}

		if pl.SkywalkY >= 0 && pl.SkywalkY < level.CHUNK_SIZE_Y {
			centerX, centerZ := int32(math.Floor(pl.X)), int32(math.Floor(pl.Z))
			for offsetX := int32(-skywalkRadius); offsetX <= skywalkRadius; offsetX++ {
				for offsetZ := int32(-skywalkRadius); offsetZ <= skywalkRadius; offsetZ++ {
					pos := [4]int32{centerX + offsetX, pl.SkywalkY, centerZ + offsetZ, pl.Dimension}
					if pl.SkywalkCarpet[pos] && world.GetBlock(pos[0], byte(pos[1]), pos[2], pos[3]).TypeId != byte(constants.Glass.Value) {
						delete(pl.SkywalkCarpet, pos)
					}
					if !pl.SkywalkCarpet[pos] {
						if world.GetBlock(pos[0], byte(pos[1]), pos[2], pos[3]).TypeId != 0 {
							continue
						}
						world.SetBlockInQueue(pos[0], pos[1], pos[2], constants.NewGlassBlock(), pos[3])
						if pl.SkywalkCarpet == nil {
							pl.SkywalkCarpet = map[[4]int32]bool{}
						}
						pl.SkywalkCarpet[pos] = true
					}
					want[pos] = true
				}
			}
		}
	}

	for pos := range pl.SkywalkCarpet {
		if want[pos] {
			continue
		}
		delete(pl.SkywalkCarpet, pos)
		if skywalkClaimed(world, pos) || world.GetBlock(pos[0], byte(pos[1]), pos[2], pos[3]).TypeId != byte(constants.Glass.Value) {
			continue
		}
		world.SetBlockInQueue(pos[0], pos[1], pos[2], constants.NewAirBlock(), pos[3])
		world.TriggerFallableUpdate(pos[0], pos[1], pos[2], world.SetBlockInQueue, pos[3])
		world.TriggerFluidUpdate(pos[0], pos[1], pos[2], world.SetBlockInQueue, pos[3])
	}

	if climbed {
		world.FlushBlockQueue()
		pl.Y = float64(pl.SkywalkY + 1)
		pl.Stance = pl.Y + playerEyeHeight
		p := packets.PlayerPositionAndRotationPacket{X: pl.X, Y: pl.Y, Stance: pl.Stance, Z: pl.Z, Yaw: pl.Yaw, Pitch: pl.Pitch, OnGround: true}
		pl.Connection.Write(p.Serialize())
	}
}

func skywalkClaimed(world *level.World, pos [4]int32) bool {
	for _, other := range world.Players {
		if other.SkywalkCarpet[pos] {
			return true
		}
	}
	return false
}

func ClearSkywalk(world *level.World, pl *player.Player) {
	pl.Skywalk = false
	pl.SkywalkFalling = true
	updateSkywalk(world, pl)
}
