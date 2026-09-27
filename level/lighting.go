package level

import "github.com/leNicDev/retromc/constants"

const (
	lightAreaXZ = 3 * CHUNK_SIZE_X
	lightAreaY  = CHUNK_SIZE_Y
)

type lightPos struct{ x, y, z int16 }

func (w *World) RelightForSend(cx, cz, dim int32, c *Chunk) {
	c.RelightAll()

	var grid [3][3]*Chunk
	for dx := -1; dx <= 1; dx++ {
		for dz := -1; dz <= 1; dz++ {
			if dx == 0 && dz == 0 {
				grid[1][1] = c
				continue
			}
			if n, ok := w.PeekChunk(cx+int32(dx), cz+int32(dz), dim); ok && len(n.Data) >= chunkBlocksAmount {
				grid[dx+1][dz+1] = n
			}
		}
	}

	blockAt := func(x, y, z int) byte {
		ch := grid[x>>4][z>>4]
		if ch == nil {
			return byte(constants.Stone.Value)
		}
		return ch.Data[(x&15)*CHUNK_SIZE_Z*CHUNK_SIZE_Y+(z&15)*CHUNK_SIZE_Y+y]
	}

	var light []byte
	var queue []lightPos
	idx := func(x, y, z int) int { return (x*lightAreaXZ+z)*lightAreaY + y }

	for gx := 0; gx < 3; gx++ {
		for gz := 0; gz < 3; gz++ {
			ch := grid[gx][gz]
			if ch == nil {
				continue
			}
			for i := 0; i < chunkBlocksAmount; i++ {
				e := constants.LightEmission[ch.Data[i]]
				if e == 0 {
					continue
				}
				lx, lz, y := i/(CHUNK_SIZE_Z*CHUNK_SIZE_Y), (i/CHUNK_SIZE_Y)%CHUNK_SIZE_Z, i%CHUNK_SIZE_Y
				x, z := gx*16+lx, gz*16+lz
				// Emitters too far away to reach the center chunk can be skipped.
				if x+int(e) < 16 || x-int(e) > 31 || z+int(e) < 16 || z-int(e) > 31 {
					continue
				}
				if light == nil {
					light = make([]byte, lightAreaXZ*lightAreaXZ*lightAreaY)
				}
				if light[idx(x, y, z)] < e {
					light[idx(x, y, z)] = e
					queue = append(queue, lightPos{int16(x), int16(y), int16(z)})
				}
			}
		}
	}

	dirs := [6][3]int{{-1, 0, 0}, {1, 0, 0}, {0, -1, 0}, {0, 1, 0}, {0, 0, -1}, {0, 0, 1}}
	for len(queue) > 0 {
		p := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		cur := light[idx(int(p.x), int(p.y), int(p.z))]
		for _, d := range dirs {
			nx, ny, nz := int(p.x)+d[0], int(p.y)+d[1], int(p.z)+d[2]
			if nx < 0 || nx >= lightAreaXZ || nz < 0 || nz >= lightAreaXZ || ny < 0 || ny >= lightAreaY {
				continue
			}
			op := constants.LightOpacity[blockAt(nx, ny, nz)]
			if op == 0 {
				op = 1
			}
			if cur <= op {
				continue
			}
			nv := cur - op
			ni := idx(nx, ny, nz)
			if light[ni] >= nv {
				continue
			}
			light[ni] = nv
			queue = append(queue, lightPos{int16(nx), int16(ny), int16(nz)})
		}
	}

	for lx := 0; lx < CHUNK_SIZE_X; lx++ {
		for lz := 0; lz < CHUNK_SIZE_Z; lz++ {
			for y := 0; y < CHUNK_SIZE_Y; y++ {
				var v byte
				if light != nil {
					v = light[idx(16+lx, y, 16+lz)]
				}
				i := lx*CHUNK_SIZE_Z*CHUNK_SIZE_Y + lz*CHUNK_SIZE_Y + y
				shift := uint((i & 1) * 4)
				mask := byte(0x0f) << shift
				ni := chunkLightOffset + i>>1
				c.Data[ni] = (c.Data[ni] &^ mask) | (v << shift)
			}
		}
	}
}
