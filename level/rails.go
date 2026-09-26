package level

import "github.com/leNicDev/retromc/constants"

const maxRailPowerDistance = 8

var sixDirs = [6][3]int32{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}}

func (w *World) torchNextTo(x, y, z, dim int32) bool {
	for _, d := range sixDirs {
		ny := y + d[1]
		if ny < 0 || ny >= CHUNK_SIZE_Y {
			continue
		}
		if w.GetBlock(x+d[0], byte(ny), z+d[2], dim).TypeId == byte(constants.RedstoneTorchOn.Value) {
			return true
		}
	}
	return false
}

func (w *World) railDirectlyPowered(x, y, z, dim int32) bool {
	return w.torchNextTo(x, y, z, dim) || (y > 0 && w.torchNextTo(x, y-1, z, dim))
}

func (w *World) poweredRailAt(x, y, z, dim int32) (constants.WBlock, bool) {
	if y < 0 || y >= CHUNK_SIZE_Y || !w.IsLoaded(x, z, dim) {
		return constants.WBlock{}, false
	}
	b := w.GetBlock(x, byte(y), z, dim)
	return b, b.IsPoweredRail()
}

func railAxis(meta byte) (dx, dz int32) {
	switch meta & 7 {
	case 0, 4, 5:
		return 0, 1
	}
	return 1, 0
}

func (w *World) connectedPoweredRails(x, y, z, dim int32) [][3]int32 {
	b, ok := w.poweredRailAt(x, y, z, dim)
	if !ok {
		return nil
	}
	dx, dz := railAxis(b.Metadata)
	var out [][3]int32
	for _, s := range []int32{-1, 1} {
		for _, dy := range []int32{0, 1, -1} {
			nx, ny, nz := x+dx*s, y+dy, z+dz*s
			if nb, ok := w.poweredRailAt(nx, ny, nz, dim); ok {
				if ndx, ndz := railAxis(nb.Metadata); ndx == dx && ndz == dz {
					out = append(out, [3]int32{nx, ny, nz})
					break
				}
			}
		}
	}
	return out
}

// UpdatePoweredRails recomputes the power bit of powered rails around a changed block
func (w *World) UpdatePoweredRails(x, y, z, dim int32) {
	const maxRails = 256

	var queue [][3]int32
	seen := map[[3]int32]bool{}
	for dx := int32(-1); dx <= 1; dx++ {
		for dy := int32(-1); dy <= 2; dy++ {
			for dz := int32(-1); dz <= 1; dz++ {
				p := [3]int32{x + dx, y + dy, z + dz}
				if _, ok := w.poweredRailAt(p[0], p[1], p[2], dim); ok && !seen[p] {
					seen[p] = true
					queue = append(queue, p)
				}
			}
		}
	}

	var rails [][3]int32
	for len(queue) > 0 && len(rails) < maxRails {
		p := queue[0]
		queue = queue[1:]
		rails = append(rails, p)
		for _, n := range w.connectedPoweredRails(p[0], p[1], p[2], dim) {
			if !seen[n] {
				seen[n] = true
				queue = append(queue, n)
			}
		}
	}

	dist := map[[3]int32]int{}
	var frontier [][3]int32
	for _, p := range rails {
		if w.railDirectlyPowered(p[0], p[1], p[2], dim) {
			dist[p] = 0
			frontier = append(frontier, p)
		}
	}
	for len(frontier) > 0 {
		p := frontier[0]
		frontier = frontier[1:]
		if dist[p] >= maxRailPowerDistance {
			continue
		}
		for _, n := range w.connectedPoweredRails(p[0], p[1], p[2], dim) {
			if _, done := dist[n]; !done && seen[n] {
				dist[n] = dist[p] + 1
				frontier = append(frontier, n)
			}
		}
	}

	for _, p := range rails {
		b, _ := w.poweredRailAt(p[0], p[1], p[2], dim)
		meta := b.Metadata &^ 0x8
		if _, powered := dist[p]; powered {
			meta |= 0x8
		}
		if meta != b.Metadata {
			b.Metadata = meta
			w.SetBlockInQueue(p[0], p[1], p[2], b, dim)
		}
	}
}
