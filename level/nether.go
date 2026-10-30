package level

import "github.com/leNicDev/retromc/constants"

const SpawnChunkRadius = 3

func (w *World) BuildLandingPad(x, y, z, dim int32, floor constants.WBlock) {
	air := constants.NewAirBlock()
	for dx := int32(-1); dx <= 1; dx++ {
		for dz := int32(-1); dz <= 1; dz++ {
			w.setBlockIfDifferent(x+dx, y-1, z+dz, dim, floor)
			for dy := int32(0); dy < 3; dy++ {
				w.setBlockIfDifferent(x+dx, y+dy, z+dz, dim, air)
			}
		}
	}
}

func (w *World) setBlockIfDifferent(x, y, z, dim int32, b constants.WBlock) {
	if w.GetBlock(x, byte(y), z, dim).TypeId != b.TypeId {
		w.SetBlockInQueue(x, y, z, b, dim)
	}
}
