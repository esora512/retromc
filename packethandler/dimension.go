package packethandler

import (
	"math"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/player"
)

const (
	netherEntryY = 6.0
	netherExitY  = 120.0
	skyEntryY    = 120.0
	skyExitY     = 4.0
)

func checkDimensionTransfer(world *level.World, pl *player.Player, prevY float64) bool {
	if pl.Transferring || pl.IsRiding != -1 {
		return false
	}
	switch {
	case pl.Dimension == 0 && pl.Y < netherEntryY:
		transferToNether(world, pl)
	case pl.Dimension == -1 && pl.Y > netherExitY:
		transferFromNether(world, pl)
	case pl.Dimension == 0 && pl.Y > skyEntryY && prevY <= skyEntryY:
		transferToSky(world, pl)
	case pl.Dimension == level.SkyDim && pl.Y < skyExitY:
		transferFromSky(world, pl)
	default:
		return false
	}
	return true
}

type landing struct {
	minY, maxY, padY int32
	surface          bool
	floor            constants.WBlock
}

const landingRadius = 24

var (
	netherLanding    = landing{minY: 10, maxY: 116, padY: 104, floor: constants.NewBlockById(constants.Netherrack.Value, 0)}
	skyLanding       = landing{minY: 6, maxY: 64, padY: level.SkyEntryY + 4, floor: constants.NewBlockById(constants.Dirt.Value, 0)}
	overworldLanding = landing{minY: netherEntryY + 2, maxY: level.CHUNK_SIZE_Y - 2, padY: 64, surface: true, floor: constants.NewBlockById(constants.Dirt.Value, 0)}
)

func transferToNether(world *level.World, pl *player.Player) {
	transferNear(world, pl, -1, netherLanding)
}

func transferFromNether(world *level.World, pl *player.Player) {
	transferNear(world, pl, 0, overworldLanding)
}

func transferToSky(world *level.World, pl *player.Player) {
	transferNear(world, pl, level.SkyDim, skyLanding)
}

func transferFromSky(world *level.World, pl *player.Player) {
	transferNear(world, pl, 0, overworldLanding)
}

func transferToOverworld(world *level.World, pl *player.Player) {
	x, y, z := overworldSpawnPoint(world, pl)
	transferPlayer(world, pl, 0, x, y, z)
}

func transferNear(world *level.World, pl *player.Player, dim int32, l landing) {
	x, z := int32(math.Floor(pl.X)), int32(math.Floor(pl.Z))
	moveToDim(world, pl, dim, float64(x)+0.5, float64(l.padY), float64(z)+0.5, func() (float64, float64, float64) {
		bx, by, bz, ok := findLanding(world, dim, x, z, l)
		if !ok {
			bx, by, bz = x, l.padY, z
			world.BuildLandingPad(bx, by, bz, dim, l.floor)
		}
		return float64(bx) + 0.5, float64(by) + 0.01, float64(bz) + 0.5
	})
}

func passable(b constants.WBlock) bool {
	switch b.TypeId {
	case byte(constants.Fire.Value), byte(constants.Cactus.Value), byte(constants.Cobweb.Value), byte(constants.NetherPortal.Value):
		return false
	}
	return !entities.IsNormalCube(b) && !b.IsLiquid()
}

func landingSpots(world *level.World, dim, x, z int32, l landing) []int32 {
	block := func(y int32) constants.WBlock { return world.GetBlock(x, byte(y), z, dim) }
	if l.surface {
		y := int32(level.CHUNK_SIZE_Y - 1)
		for y > 0 && passable(block(y)) {
			y--
		}
		if y+1 >= l.minY && y+1 <= l.maxY && entities.IsNormalCube(block(y)) {
			return []int32{y + 1}
		}
		return nil
	}
	var spots []int32
	for y := l.minY; y <= l.maxY; y++ {
		if entities.IsNormalCube(block(y-1)) && passable(block(y)) && passable(block(y+1)) {
			spots = append(spots, y)
		}
	}
	return spots
}

func findLanding(world *level.World, dim, x, z int32, l landing) (int32, int32, int32, bool) {
	var best [3]int32
	bestScore := int32(math.MaxInt32)
	for r := int32(0); r <= landingRadius && r*r < bestScore; r++ {
		for dx := -r; dx <= r; dx++ {
			for dz := -r; dz <= r; dz++ {
				if max(abs32(dx), abs32(dz)) != r {
					continue
				}
				for _, y := range landingSpots(world, dim, x+dx, z+dz, l) {
					score := dx*dx + dz*dz
					if !l.surface {
						score += (y - l.padY) * (y - l.padY)
					}
					if score < bestScore {
						best, bestScore = [3]int32{x + dx, y, z + dz}, score
					}
				}
			}
		}
	}
	return best[0], best[1], best[2], bestScore != math.MaxInt32
}

func overworldSpawnPoint(world *level.World, pl *player.Player) (float64, float64, float64) {
	if pl.HasBedSpawn {
		bed := world.GetBlock(pl.BedSpawnX, pl.BedSpawnY, pl.BedSpawnZ, 0)
		if bed.IsBed() {
			return float64(pl.BedSpawnX) + 0.5, float64(pl.BedSpawnY) + 1, float64(pl.BedSpawnZ) + 0.5
		}
		pl.HasBedSpawn = false
		sendDebugMessage(pl, "Your home bed was missing or obstructed")
	}
	return player.SpawnX, player.SpawnY, player.SpawnZ
}

func transferPlayer(world *level.World, pl *player.Player, dim int32, x, y, z float64) {
	moveToDim(world, pl, dim, x, y, z, func() (float64, float64, float64) {
		return x, freeSpawnY(world, dim, x, y, z) + 0.01, z
	})
}

func moveToDim(world *level.World, pl *player.Player, dim int32, x, y, z float64, locate func() (float64, float64, float64)) {
	pl.Transferring = true
	pl.AwaitingTeleportAck = false
	pl.Dimension = dim
	pl.X, pl.Y, pl.Z = x, y, z
	pl.Stance = y + playerEyeHeight
	pl.Yaw, pl.Pitch = 0, 0
	pl.OnGround = true
	pl.FallDistance = 0
	pl.Immune = 200
	pl.SentChunksMu.Lock()
	pl.SentChunks = make(player.ChunkSet)
	pl.SentChunksMu.Unlock()
	pl.RequestedChunks = make(player.ChunkSet)
	pl.PendingChunks = make(player.ChunkSet)
	pl.HasInitializedChunks = false

	sendRespawn(pl.Connection, byte(dim))

	cx := level.WorldToChunkCoord(int32(math.Floor(x)))
	cz := level.WorldToChunkCoord(int32(math.Floor(z)))
	ensureChunksLoaded(world, dim, cx, cz, func() {
		if !world.HasPlayer(pl) || pl.Dimension != dim {
			return
		}
		x, y, z := locate()
		pl.X, pl.Y, pl.Z = x, y, z
		pl.Stance = y + playerEyeHeight

		pl.SentChunksMu.Lock()
		forSpawnChunks(cx, cz, func(coord level.ChunkCoord) {
			if chunk, ok := world.PeekChunk(coord.X, coord.Z, dim); ok {
				sendChunkToPlayer(world, pl, coord, dim, chunk)
			}
		})
		pl.SentChunksMu.Unlock()
		sendPlayerPositionAndLook(pl.Connection, x, z, y)
		sendInventory(pl.Connection, pl, world)
		applyChunkVisibility(world, pl, cx, cz)
		pl.LastChunkX, pl.LastChunkZ, pl.LastDim = cx, cz, dim
		pl.HasInitializedChunks = true
		pl.MovementState.Teleported = true
		pl.AwaitingTeleportAck = true
	})
}

func freeSpawnY(world *level.World, dim int32, x, y, z float64) float64 {
	bx, bz := int32(math.Floor(x)), int32(math.Floor(z))
	blocked := func(by int32) bool {
		return entities.IsNormalCube(world.GetBlock(bx, byte(by), bz, dim))
	}
	start := max(int32(math.Floor(y)), 0)
	for by := start; by < level.CHUNK_SIZE_Y-1; by++ {
		if !blocked(by) && !blocked(by+1) {
			if by == start {
				return y
			}
			return float64(by)
		}
	}
	return y
}

func acceptsMovement(pl *player.Player, x, z float64) bool {
	if !pl.Transferring {
		return true
	}
	if pl.AwaitingTeleportAck && math.Abs(x-pl.X) < 0.01 && math.Abs(z-pl.Z) < 0.01 {
		pl.Transferring = false
		pl.AwaitingTeleportAck = false
		return true
	}
	return false
}

func forSpawnChunks(cx, cz int32, f func(level.ChunkCoord)) {
	for dx := int32(-level.SpawnChunkRadius); dx <= level.SpawnChunkRadius; dx++ {
		for dz := int32(-level.SpawnChunkRadius); dz <= level.SpawnChunkRadius; dz++ {
			f(level.ChunkCoord{X: cx + dx, Z: cz + dz})
		}
	}
}

func ensureChunksLoaded(world *level.World, dim, cx, cz int32, onReady func()) {
	remaining := 0
	forSpawnChunks(cx, cz, func(c level.ChunkCoord) {
		if world.ChunkExists(c.X, c.Z, dim) {
			return
		}
		remaining++
		world.RequestChunkAsync(c.X, c.Z, dim, func(generated *level.Chunk) {
			world.Enqueue(func() {
				world.InsertChunk(c.X, c.Z, dim, generated)
				remaining--
				if remaining == 0 {
					onReady()
				}
			})
		})
	})
	if remaining == 0 {
		onReady()
	}
}
