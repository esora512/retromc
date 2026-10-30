package level

import (
	"math"
	"path/filepath"

	"github.com/leNicDev/retromc/constants"
)

var Dims = []int32{0, -1, SkyDim}

func dimDir(worldDir string, dim int32) string {
	switch dim {
	case -1:
		return filepath.Join(worldDir, "DIM-1")
	case SkyDim:
		return filepath.Join(worldDir, "DIM1")
	}
	return worldDir
}

const (
	SkyDim    = 1
	SkyEntryY = 8
)

const (
	vaultCell  = 48
	vaultMaxR  = 12
	vaultKinds = 18
)

var vaultCores = []byte{
	byte(constants.IronBlock.Value), byte(constants.DiamondBlock.Value), byte(constants.GoldBlock.Value),
	idRedstoneOr, byte(constants.LapisLazuliBlock.Value),
}

func inPolygon(sides int, r, x, z float64) bool {
	if sides == 0 {
		return x*x+z*z <= r*r
	}
	sector := 2 * math.Pi / float64(sides)
	a := math.Mod(math.Atan2(z, x)+2*math.Pi, sector) - sector/2
	return math.Hypot(x, z)*math.Cos(a) <= r*math.Cos(math.Pi/float64(sides))
}

func vaultShape(kind, s, dx, dy, dz int) bool {
	ax, ay, az := absIn(dx), absIn(dy), absIn(dz)
	r, x, y, z, h := float64(s)+0.5, float64(dx), float64(dy), float64(dz), float64(s)
	switch kind {
	case 15:
		return x*x+y*y+z*z <= r*r
	case 16:
		return max(ax, ay, az) <= s
	case 17:
		return ax+ay+az-max(ax, ay, az)-min(ax, ay, az) <= 1 && max(ax, ay, az) <= s+1
	}
	if ay > s {
		return false
	}
	scale := 1.0
	switch kind / 5 {
	case 1:
		scale = 1 - math.Abs(y)/(h+1)
	case 2:
		scale = (h + 1 - y) / (2*h + 1) * 1.6
	}
	return inPolygon([]int{0, 4, 5, 6, 8}[kind%5], r*scale, x, z)
}

func carveSkyVaults(blocks []byte, worldX, worldZ int, seed uint32) {
	cell := func(v int) int { return int(math.Floor(float64(v) / vaultCell)) }
	for cellX := cell(worldX - vaultMaxR); cellX <= cell(worldX+CHUNK_SIZE_X+vaultMaxR); cellX++ {
		for cellZ := cell(worldZ - vaultMaxR); cellZ <= cell(worldZ+CHUNK_SIZE_Z+vaultMaxR); cellZ++ {
			rnd := chunkRand(int64(seed)^0x5a17, int32(cellX), int32(cellZ))
			if rnd.Intn(3) == 0 {
				continue
			}
			s := 3 + rnd.Intn(3)
			r := s + 5 + rnd.Intn(vaultMaxR-s-4)
			cx := cellX*vaultCell + vaultMaxR + rnd.Intn(vaultCell-2*vaultMaxR)
			cz := cellZ*vaultCell + vaultMaxR + rnd.Intn(vaultCell-2*vaultMaxR)
			cy := SkyEntryY + 24 + rnd.Intn(25)
			kind, core := rnd.Intn(vaultKinds), vaultCores[rnd.Intn(len(vaultCores))]
			for x := range CHUNK_SIZE_X {
				for z := range CHUNK_SIZE_Z {
					for y := cy - r; y <= cy+r; y++ {
						dx, dy, dz := worldX+x-cx, y-cy, worldZ+z-cz
						i := x*CHUNK_SIZE_Z*CHUNK_SIZE_Y + z*CHUNK_SIZE_Y + y
						switch {
						case dx == 0 && dy == 0 && dz == 0:
							blocks[i] = core
						case vaultShape(kind, s, dx, dy, dz):
							blocks[i] = byte(constants.Obsidian.Value)
						case dx*dx+dy*dy+dz*dz <= r*r:
							blocks[i] = idAir
						}
					}
				}
			}
		}
	}
}
