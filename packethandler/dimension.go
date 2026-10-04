package packethandler

import (
	"math"

	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/player"
)

const (
	netherEntryY = 6.0
	netherExitY  = 120.0
)

func checkDimensionTransfer(world *level.World, pl *player.Player) bool {
	if pl.Transferring || pl.IsRiding != -1 {
		return false
	}
	switch {
	case pl.Dimension == 0 && pl.Y < netherEntryY:
		transferToNether(world, pl)
	case pl.Dimension == -1 && pl.Y > netherExitY:
		transferToOverworld(world, pl)
	default:
		return false
	}
	return true
}

func transferToNether(world *level.World, pl *player.Player) {
	world.BuildNetherLandingPad()
	transferPlayer(world, pl, -1, level.NetherSpawnX+0.5, level.NetherSpawnY, level.NetherSpawnZ+0.5)
}

func transferToOverworld(world *level.World, pl *player.Player) {
	x, y, z := overworldSpawnPoint(world, pl)
	transferPlayer(world, pl, 0, x, y, z)
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
		y := freeSpawnY(world, dim, x, y, z) + 0.01
		pl.Y = y
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
