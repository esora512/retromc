package level

// This code is AI-generated for experimental world gen purposes; based on: https://www.planetminecraft.com/data-pack/noodle-world-generation-experiment/

// GenerateNoodleWorld: terrain is built from two independent 3D noise
// fields. A block is solid only where BOTH fields are close to zero -
// this is the same trick Minecraft's real noodle caves use. One field
// alone defines a wobbly sheet in 3D space; intersecting two sheets
// collapses down to a winding 1D tube, so you get thick worming noodles
// instead of flat ground. Grass is added wherever solid touches air above.

import "math"

// hash3 returns a pseudo-random float in [-1,1] for an integer lattice
// point, mixed with the seed via splitmix64 (defined in worldgen_bareiron.go).
func hash3(ix, iy, iz int64, seed uint32) float64 {
	h := uint64(ix) * 0x9E3779B97F4A7C15
	h ^= uint64(iy) * 0xC2B2AE3D27D4EB4F
	h ^= uint64(iz) * 0x165667B19E3779F9
	h ^= uint64(seed) * 0x27D4EB2F165667C5
	h = splitmix64(h)
	return (float64(h%1_000_000)/1_000_000.0)*2 - 1
}

func smoothstep(t float64) float64 { return t * t * (3 - 2*t) }
func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// valueNoise3D samples a smooth pseudo-random field at (x,y,z).
func valueNoise3D(x, y, z float64, seed uint32) float64 {
	x0, y0, z0 := math.Floor(x), math.Floor(y), math.Floor(z)
	ix0, iy0, iz0 := int64(x0), int64(y0), int64(z0)
	fx, fy, fz := smoothstep(x-x0), smoothstep(y-y0), smoothstep(z-z0)

	c000 := hash3(ix0, iy0, iz0, seed)
	c100 := hash3(ix0+1, iy0, iz0, seed)
	c010 := hash3(ix0, iy0+1, iz0, seed)
	c110 := hash3(ix0+1, iy0+1, iz0, seed)
	c001 := hash3(ix0, iy0, iz0+1, seed)
	c101 := hash3(ix0+1, iy0, iz0+1, seed)
	c011 := hash3(ix0, iy0+1, iz0+1, seed)
	c111 := hash3(ix0+1, iy0+1, iz0+1, seed)

	x00 := lerp(c000, c100, fx)
	x10 := lerp(c010, c110, fx)
	x01 := lerp(c001, c101, fx)
	x11 := lerp(c011, c111, fx)

	y0v := lerp(x00, x10, fy)
	y1v := lerp(x01, x11, fy)

	return lerp(y0v, y1v, fz)
}

const (
	noodleFreq1      = 0.06
	noodleFreq2      = noodleFreq1 * 1.618 // golden-ratio offset decorrelates the two fields
	noodleThickness1 = 0.18                // raise for fatter noodles, lower for thinner/sparser
	noodleThickness2 = 0.18
)

// GenerateNoodleWorld fills the chunk based on the two-field intersection
// test described above, then caps any solid block with air above it in grass.
func (c *Chunk) GenerateNoodleWorld(seed uint32, cx, cz int32) {
	blocksAmount := CHUNK_SIZE_X * CHUNK_SIZE_Y * CHUNK_SIZE_Z
	nibbleCount := blocksAmount / 2

	blockTypes := make([]byte, blocksAmount)
	blockMetadata := make([]byte, nibbleCount)
	blockLight := make([]byte, nibbleCount)
	blockSkyLight := make([]byte, nibbleCount)

	worldX := int(cx) * CHUNK_SIZE_X
	worldZ := int(cz) * CHUNK_SIZE_Z

	// Seed-derived pan offset so each seed explores a different region
	// of the noise field (otherwise every seed would look identical).
	h1 := splitmix64(uint64(seed))
	h2 := splitmix64(h1)
	h3 := splitmix64(h2)
	panX := float64(h1%1_000_000) / 1_000_000.0 * 5000
	panY := float64(h2%1_000_000) / 1_000_000.0 * 5000
	panZ := float64(h3%1_000_000) / 1_000_000.0 * 5000

	for i := 0; i < blocksAmount; i++ {
		y := i % CHUNK_SIZE_Y
		z := (i / CHUNK_SIZE_Y) % CHUNK_SIZE_Z
		x := i / (CHUNK_SIZE_Y * CHUNK_SIZE_Z)

		wx := float64(worldX+x) + panX
		wy := float64(y) + panY
		wz := float64(worldZ+z) + panZ

		d1 := valueNoise3D(wx*noodleFreq1, wy*noodleFreq1, wz*noodleFreq1, seed)
		d2 := valueNoise3D(wx*noodleFreq2, wy*noodleFreq2, wz*noodleFreq2, seed+1)

		if math.Abs(d1) < noodleThickness1 && math.Abs(d2) < noodleThickness2 {
			blockTypes[i] = idStone
		} else {
			blockTypes[i] = idAir
		}
	}

	// Grass cap: any stone block with air directly above becomes grass.
	// Because noodles wind through space, a single column can get several
	// grass caps stacked at different heights - that's expected and looks
	// like the reference (mossy tube surfaces facing "up" everywhere).
	for x := 0; x < CHUNK_SIZE_X; x++ {
		for z := 0; z < CHUNK_SIZE_Z; z++ {
			for y := 0; y < CHUNK_SIZE_Y; y++ {
				i := x*CHUNK_SIZE_Z*CHUNK_SIZE_Y + z*CHUNK_SIZE_Y + y
				if blockTypes[i] != idStone {
					continue
				}
				aboveId := byte(idAir)
				if y+1 < CHUNK_SIZE_Y {
					aboveId = blockTypes[x*CHUNK_SIZE_Z*CHUNK_SIZE_Y+z*CHUNK_SIZE_Y+(y+1)]
				}
				if aboveId == idAir {
					blockTypes[i] = idGrass
				}
			}
		}
	}

	c.Data = blockTypes
	c.Data = append(c.Data, blockMetadata...)
	c.Data = append(c.Data, blockLight...)
	c.Data = append(c.Data, blockSkyLight...)
	c.RelightAll()
}
