package level

import (
	"math"
	"math/rand"
)

type PerlinNoise struct {
	perm [512]int
}

func NewPerlinNoise(seed int64) *PerlinNoise {
	p := &PerlinNoise{}
	permutation := make([]int, 256)
	for i := range permutation {
		permutation[i] = i
	}
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(permutation), func(i, j int) {
		permutation[i], permutation[j] = permutation[j], permutation[i]
	})
	for i := 0; i < 512; i++ {
		p.perm[i] = permutation[i%256]
	}
	return p
}

func fade(t float64) float64       { return t * t * t * (t*(t*6-15) + 10) }
func lerp(t, a, b float64) float64 { return a + t*(b-a) }

func grad(hash int, x, y float64) float64 {
	switch hash & 3 {
	case 0:
		return x + y
	case 1:
		return -x + y
	case 2:
		return x - y
	default:
		return -x - y
	}
}

func (p *PerlinNoise) Noise2D(x, y float64) float64 {
	X := int(math.Floor(x)) & 255
	Y := int(math.Floor(y)) & 255
	xf := x - math.Floor(x)
	yf := y - math.Floor(y)
	u, v := fade(xf), fade(yf)

	aa := p.perm[X+p.perm[Y]]
	ab := p.perm[X+p.perm[Y+1]]
	ba := p.perm[X+1+p.perm[Y]]
	bb := p.perm[X+1+p.perm[Y+1]]

	x1 := lerp(u, grad(aa, xf, yf), grad(ba, xf-1, yf))
	x2 := lerp(u, grad(ab, xf, yf-1), grad(bb, xf-1, yf-1))
	return lerp(v, x1, x2)
}

func (p *PerlinNoise) OctaveNoise2D(x, y float64, octaves int, persistence float64) float64 {
	total, frequency, amplitude, maxValue := 0.0, 1.0, 1.0, 0.0
	for i := 0; i < octaves; i++ {
		total += p.Noise2D(x*frequency, y*frequency) * amplitude
		maxValue += amplitude
		amplitude *= persistence
		frequency *= 2
	}
	return total / maxValue // normalized to roughly [-1, 1]
}

func (c *Chunk) initData() {
	blocksAmount := CHUNK_SIZE_X * CHUNK_SIZE_Y * CHUNK_SIZE_Z
	nibbleCount := blocksAmount / 2
	c.Data = make([]byte, blocksAmount+3*nibbleCount)
}

func (c *Chunk) GenerateTerrain(noise *PerlinNoise, worldX, worldZ int32) {
	c.initData()

	for lx := int32(0); lx < CHUNK_SIZE_X; lx++ {
		for lz := int32(0); lz < CHUNK_SIZE_Z; lz++ {
			wx := float64(worldX + lx)
			wz := float64(worldZ + lz)

			n := noise.OctaveNoise2D(wx/64.0, wz/64.0, 4, 0.5)
			height := int32((n + 1) / 2 * float64(CHUNK_SIZE_Y-1))

			for ly := int32(0); ly <= height; ly++ {
				block := NewStoneBlock()
				if ly == height {
					block = NewGrassBlock()
					block.SkyLight = 0x0f
				} else if ly > height-4 {
					block = NewDirtBlock() 
				}
				c.SetBlock(int(lx), int(ly), int(lz), block)
			}
		}
	}
	c.relightAll()
}
