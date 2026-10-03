package level

import "github.com/leNicDev/retromc/constants"

const (
	NetherSpawnX      = 0
	NetherSpawnY      = 9
	NetherSpawnZ      = 0
	SpawnChunkRadius  = 3
	netherSpawnRadius = SpawnChunkRadius
)

func netherSpawnChunk() ChunkCoord {
	return ChunkCoord{WorldToChunkCoord(NetherSpawnX), WorldToChunkCoord(NetherSpawnZ)}
}

func addNetherSpawnChunks(wanted map[ChunkCoord]struct{}) {
	c := netherSpawnChunk()
	for dx := int32(-netherSpawnRadius); dx <= netherSpawnRadius; dx++ {
		for dz := int32(-netherSpawnRadius); dz <= netherSpawnRadius; dz++ {
			wanted[ChunkCoord{X: c.X + dx, Z: c.Z + dz}] = struct{}{}
		}
	}
}

func (w *World) PrepareNetherSpawn() {
	wanted := make(map[ChunkCoord]struct{})
	addNetherSpawnChunks(wanted)
	for c := range wanted {
		w.GetOrCreateChunk(c.X, c.Z, -1)
	}
	w.BuildNetherLandingPad()
}

func (w *World) BuildNetherLandingPad() {
	rack := constants.NewBlockById(constants.Netherrack.Value, 0)
	air := constants.NewAirBlock()
	for dx := int32(-1); dx <= 1; dx++ {
		for dz := int32(-1); dz <= 1; dz++ {
			x, z := NetherSpawnX+dx, NetherSpawnZ+dz
			w.setBlockIfDifferent(x, NetherSpawnY-1, z, rack)
			for dy := int32(0); dy < 3; dy++ {
				w.setBlockIfDifferent(x, NetherSpawnY+dy, z, air)
			}
		}
	}
}

func (w *World) setBlockIfDifferent(x, y, z int32, b constants.WBlock) {
	if w.GetBlock(x, byte(y), z, -1).TypeId != b.TypeId {
		w.SetBlockInQueue(x, y, z, b, -1)
	}
}
