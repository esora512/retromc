package level

import (
	"sync"

	"github.com/leNicDev/retromc/constants"
)

const (
	lightSectionHeight = 16
	lightSections      = CHUNK_SIZE_Y / lightSectionHeight
	lightSectionBytes  = CHUNK_SIZE_X * CHUNK_SIZE_Z * lightSectionHeight / 2
)

// A lightSection without data holds the same value in every block.
type lightSection struct {
	data []byte
	fill byte
}

type lightLayer [lightSections]lightSection

func sectionIndex(lx, y, lz int) int { return lx<<8 | lz<<4 | y&15 }

func (l *lightLayer) get(lx, y, lz int) byte {
	s := &l[y>>4]
	if s.data == nil {
		return s.fill
	}
	return getNibble(s.data, sectionIndex(lx, y, lz))
}

func (l *lightLayer) set(lx, y, lz int, v byte) {
	s := &l[y>>4]
	if s.data == nil {
		if v == s.fill {
			return
		}
		s.data = make([]byte, lightSectionBytes)
		if s.fill != 0 {
			b := s.fill * 0x11
			for i := range s.data {
				s.data[i] = b
			}
		}
	}
	setNibble(s.data, sectionIndex(lx, y, lz), v)
}

// pairAt returns the nibbles of y and y+1 packed into one byte; y must be even.
func (l *lightLayer) pairAt(lx, y, lz int) byte {
	s := &l[y>>4]
	if s.data == nil {
		return s.fill * 0x11
	}
	return s.data[sectionIndex(lx, y, lz)>>1]
}

func (l *lightLayer) compact() {
	for i := range l {
		s := &l[i]
		if s.data == nil {
			continue
		}
		b := s.data[0]
		if b&15 != b>>4 {
			continue
		}
		uniform := true
		for _, v := range s.data {
			if v != b {
				uniform = false
				break
			}
		}
		if uniform {
			s.data, s.fill = nil, b&15
		}
	}
}

func (l *lightLayer) size() int64 {
	var n int64
	for i := range l {
		n += int64(len(l[i].data))
	}
	return n
}

// region writes the box in packet and NBT order (x, then z, then y); y0 must be even and y1 odd.
func (l *lightLayer) region(dst []byte, x0, y0, z0, x1, y1, z1 int) int {
	n := 0
	for x := x0; x <= x1; x++ {
		for z := z0; z <= z1; z++ {
			for y := y0; y < y1; y += 2 {
				dst[n] = l.pairAt(x, y, z)
				n++
			}
		}
	}
	return n
}

func (l *lightLayer) load(flat []byte) {
	const columns = CHUNK_SIZE_X * CHUNK_SIZE_Z
	const colBytes = CHUNK_SIZE_Y / 2
	const secBytes = lightSectionHeight / 2
	for s := range l {
		first := flat[s*secBytes]
		uniform := first&15 == first>>4
		for col := 0; uniform && col < columns; col++ {
			for _, v := range flat[col*colBytes+s*secBytes : col*colBytes+(s+1)*secBytes] {
				if v != first {
					uniform = false
					break
				}
			}
		}
		if uniform {
			l[s] = lightSection{fill: first & 15}
			continue
		}
		data := make([]byte, lightSectionBytes)
		for col := 0; col < columns; col++ {
			copy(data[col*secBytes:(col+1)*secBytes], flat[col*colBytes+s*secBytes:])
		}
		l[s] = lightSection{data: data}
	}
}

var lightBufPool = sync.Pool{New: func() any { return new([chunkNibbleCount]byte) }}

type lightBox struct {
	ok                     bool
	x0, y0, z0, x1, y1, z1 int
}

func (b *lightBox) add(lx, y, lz int) {
	if !b.ok {
		*b = lightBox{true, lx, y, lz, lx, y, lz}
		return
	}
	b.x0, b.y0, b.z0 = min(b.x0, lx), min(b.y0, y), min(b.z0, lz)
	b.x1, b.y1, b.z1 = max(b.x1, lx), max(b.y1, y), max(b.z1, lz)
}

type dirtyLight struct {
	c   *Chunk
	dim int32
}

type lightSeeds struct {
	sky, block []lightPos
}

type lightPos struct {
	x, z int32
	y    int
}

type lightNode struct {
	p lightPos
	v byte
}

var lightDirs = [6][3]int32{{-1, 0, 0}, {1, 0, 0}, {0, -1, 0}, {0, 1, 0}, {0, 0, -1}, {0, 0, 1}}

func blockIndex(lx, y, lz int) int { return lx*CHUNK_SIZE_Z*CHUNK_SIZE_Y + lz*CHUNK_SIZE_Y + y }

func setNibble(arr []byte, i int, v byte) {
	shift := uint((i & 1) * 4)
	mask := byte(0x0f) << shift
	arr[i>>1] = (arr[i>>1] &^ mask) | (v << shift)
}

func getNibble(arr []byte, i int) byte {
	return (arr[i>>1] >> uint((i&1)*4)) & 0x0f
}

func lightCost(id byte) byte { return max(1, constants.LightOpacity[id]) }

func (c *Chunk) columnHeight(lx, lz int) byte {
	base := blockIndex(lx, 0, lz)
	for y := CHUNK_SIZE_Y - 1; y >= 0; y-- {
		if constants.LightOpacity[c.Data[base+y]] > 0 {
			return byte(y + 1)
		}
	}
	return 0
}

func (c *Chunk) computeHeight() {
	for lx := 0; lx < CHUNK_SIZE_X; lx++ {
		for lz := 0; lz < CHUNK_SIZE_Z; lz++ {
			c.Height[lz*CHUNK_SIZE_X+lx] = c.columnHeight(lx, lz)
		}
	}
}

type cachedChunk struct {
	ok     bool
	cx, cz int32
	c      *Chunk
}

type lighter struct {
	w       *World
	dim     int32
	sky     bool
	persist bool
	buckets [16][]lightPos

	cache [4]cachedChunk
	next  int
}

func (l *lighter) chunk(x, z int32) *Chunk {
	cx, cz := x>>4, z>>4
	for i := range l.cache {
		if e := &l.cache[i]; e.ok && e.cx == cx && e.cz == cz {
			return e.c
		}
	}
	c, _ := l.w.PeekChunk(cx, cz, l.dim)
	l.cache[l.next] = cachedChunk{true, cx, cz, c}
	l.next = (l.next + 1) % len(l.cache)
	return c
}

func (l *lighter) layer(c *Chunk) *lightLayer {
	if l.sky {
		return &c.skyLight
	}
	return &c.blockLight
}

func (l *lighter) get(c *Chunk, p lightPos) byte {
	return l.layer(c).get(int(p.x&15), p.y, int(p.z&15))
}

func (l *lighter) set(c *Chunk, p lightPos, v byte) {
	lx, lz := int(p.x&15), int(p.z&15)
	layer := l.layer(c)
	if layer.get(lx, p.y, lz) == v {
		return
	}
	layer.set(lx, p.y, lz, v)
	if l.persist {
		c.HasChanged = true
	}
	if !c.lightDirty.ok {
		l.w.dirtyLight = append(l.w.dirtyLight, dirtyLight{c, l.dim})
	}
	c.lightDirty.add(lx, p.y, lz)
}

func (l *lighter) base(c *Chunk, p lightPos) byte {
	lx, lz := int(p.x&15), int(p.z&15)
	if l.sky {
		if p.y >= int(c.Height[lz*CHUNK_SIZE_X+lx]) {
			return 15
		}
		return 0
	}
	return constants.LightEmission[c.Data[blockIndex(lx, p.y, lz)]]
}

func (l *lighter) spreadTo(v byte, p lightPos) {
	if p.y < 0 || p.y >= CHUNK_SIZE_Y {
		return
	}
	c := l.chunk(p.x, p.z)
	if c == nil {
		return
	}
	cost := lightCost(c.Data[blockIndex(int(p.x&15), p.y, int(p.z&15))])
	if v <= cost || l.get(c, p) >= v-cost {
		return
	}
	l.set(c, p, v-cost)
	l.buckets[v-cost] = append(l.buckets[v-cost], p)
}

func (l *lighter) fill() {
	for cur := byte(15); cur > 1; cur-- {
		for i := 0; i < len(l.buckets[cur]); i++ {
			p := l.buckets[cur][i]
			if l.get(l.chunk(p.x, p.z), p) != cur {
				continue
			}
			for _, d := range lightDirs {
				l.spreadTo(cur, lightPos{p.x + d[0], p.z + d[2], p.y + int(d[1])})
			}
		}
		l.buckets[cur] = nil
	}
	l.buckets[1] = nil
}

func (w *World) initChunkLight(cx, cz, dim int32, c *Chunk) {
	w.ApplyLight()
	c.needsLight = false
	c.computeHeight()

	maxH := 0
	for _, h := range c.Height {
		maxH = max(maxH, int(h))
	}
	for s := range lightSections {
		c.skyLight[s] = lightSection{}
		c.blockLight[s] = lightSection{}
		if s*lightSectionHeight >= maxH {
			c.skyLight[s].fill = 15
		}
	}
	skyTop := min(CHUNK_SIZE_Y, (maxH+lightSectionHeight-1)&^(lightSectionHeight-1))

	sky := &lighter{w: w, dim: dim, sky: true}
	blk := &lighter{w: w, dim: dim}
	ox, oz := cx*CHUNK_SIZE_X, cz*CHUNK_SIZE_Z

	for lx := 0; lx < CHUNK_SIZE_X; lx++ {
		for lz := 0; lz < CHUNK_SIZE_Z; lz++ {
			for y := int(c.Height[lz*CHUNK_SIZE_X+lx]); y < skyTop; y++ {
				c.skyLight.set(lx, y, lz, 15)
			}
			base := blockIndex(lx, 0, lz)
			for y, id := range c.Data[base : base+CHUNK_SIZE_Y] {
				if e := constants.LightEmission[id]; e > 0 {
					c.blockLight.set(lx, y, lz, e)
					blk.buckets[e] = append(blk.buckets[e], lightPos{ox + int32(lx), oz + int32(lz), y})
				}
			}
		}
	}

	for lx := 0; lx < CHUNK_SIZE_X; lx++ {
		for lz := 0; lz < CHUNK_SIZE_Z; lz++ {
			h := int(c.Height[lz*CHUNK_SIZE_X+lx])
			if h == 0 {
				continue
			}
			lo := h - 1
			for _, d := range lightDirs {
				nx, nz := lx+int(d[0]), lz+int(d[2])
				if d[1] != 0 || nx < 0 || nx >= CHUNK_SIZE_X || nz < 0 || nz >= CHUNK_SIZE_Z {
					continue
				}
				lo = min(lo, int(c.Height[nz*CHUNK_SIZE_X+nx]))
			}
			for y := lo; y < h; y++ {
				sky.spreadTo(15, lightPos{ox + int32(lx), oz + int32(lz), y})
			}
		}
	}

	for _, d := range lightDirs {
		if d[1] != 0 {
			continue
		}
		n, ok := w.PeekChunk(cx+d[0], cz+d[2], dim)
		if !ok || n.needsLight {
			continue
		}
		for i := int32(0); i < CHUNK_SIZE_X; i++ {
			var a lightPos
			switch {
			case d[0] < 0:
				a = lightPos{ox, oz + i, 0}
			case d[0] > 0:
				a = lightPos{ox + 15, oz + i, 0}
			case d[2] < 0:
				a = lightPos{ox + i, oz, 0}
			default:
				a = lightPos{ox + i, oz + 15, 0}
			}
			for y := 0; y < CHUNK_SIZE_Y; y++ {
				a.y = y
				b := lightPos{a.x + d[0], a.z + d[2], y}
				for _, l := range []*lighter{sky, blk} {
					va, vb := l.get(c, a), l.get(n, b)
					if va > vb+1 {
						l.spreadTo(va, b)
					} else if vb > va+1 {
						l.spreadTo(vb, a)
					}
				}
			}
		}
	}

	sky.fill()
	blk.fill()
	c.lightDirty = lightBox{}
	c.skyLight.compact()
	c.blockLight.compact()
}

func (l *lighter) update(seeds []lightPos) {
	queue := make([]lightNode, 0, len(seeds))
	var refill []lightPos
	reset := func(c *Chunk, p lightPos) {
		b := l.base(c, p)
		l.set(c, p, b)
		if b > 0 {
			refill = append(refill, p)
		}
	}
	for _, p := range seeds {
		c := l.chunk(p.x, p.z)
		if c == nil {
			continue
		}
		queue = append(queue, lightNode{p, l.get(c, p)})
		reset(c, p)
	}
	for i := 0; i < len(queue); i++ {
		n := queue[i]
		for _, d := range lightDirs {
			q := lightPos{n.p.x + d[0], n.p.z + d[2], n.p.y + int(d[1])}
			if q.y < 0 || q.y >= CHUNK_SIZE_Y {
				continue
			}
			c := l.chunk(q.x, q.z)
			if c == nil {
				continue
			}
			v := l.get(c, q)
			if v == 0 {
				continue
			}
			if v >= n.v {
				refill = append(refill, q)
			} else {
				queue = append(queue, lightNode{q, v})
				reset(c, q)
			}
		}
	}
	for _, p := range refill {
		if v := l.get(l.chunk(p.x, p.z), p); v > 0 {
			l.buckets[v] = append(l.buckets[v], p)
		}
	}
	l.fill()
}

// queueLight records a block change; its light is updated by the next ApplyLight.
func (w *World) queueLight(c *Chunk, x int32, y int, z int32, dim int32, oldId, newId byte) {
	opChanged := constants.LightOpacity[oldId] != constants.LightOpacity[newId]
	if !opChanged && constants.LightEmission[oldId] == constants.LightEmission[newId] {
		return
	}
	if w.lightSeeds == nil {
		w.lightSeeds = make(map[int32]*lightSeeds)
	}
	q := w.lightSeeds[dim]
	if q == nil {
		q = &lightSeeds{}
		w.lightSeeds[dim] = q
	}
	pos := lightPos{x, z, y}
	q.block = append(q.block, pos)
	if !opChanged {
		return
	}

	col := int(z&15)*CHUNK_SIZE_X + int(x&15)
	oldH := int(c.Height[col])
	c.Height[col] = c.columnHeight(int(x&15), int(z&15))
	newH := int(c.Height[col])

	q.sky = append(q.sky, pos)
	for sy := min(oldH, newH); sy < max(oldH, newH); sy++ {
		if sy != y {
			q.sky = append(q.sky, lightPos{x, z, sy})
		}
	}
}

// ApplyLight brings the stored light up to date with every block changed since the last call.
func (w *World) ApplyLight() {
	for dim, q := range w.lightSeeds {
		if len(q.block) > 0 {
			seeds := q.block
			q.block = nil
			(&lighter{w: w, dim: dim, persist: true}).update(seeds)
		}
		if len(q.sky) > 0 {
			seeds := q.sky
			q.sky = nil
			(&lighter{w: w, dim: dim, sky: true, persist: true}).update(seeds)
		}
	}
}

// FlushLight applies pending light changes and sends each changed region to the players that have its chunk.
func (w *World) FlushLight() {
	w.ApplyLight()
	dirty := w.dirtyLight
	w.dirtyLight = nil
	for _, d := range dirty {
		c, b := d.c, d.c.lightDirty
		if !b.ok {
			continue
		}
		c.lightDirty = lightBox{}
		c.skyLight.compact()
		c.blockLight.compact()
		if cur, ok := w.PeekChunk(c.X>>4, c.Z>>4, d.dim); !ok || cur != c || w.sendLightRegion == nil {
			continue
		}
		w.sendLightRegion(w, c, d.dim, b.x0, b.y0&^1, b.z0, b.x1, b.y1|1, b.z1)
	}
}

func (w *World) SetSendLightRegion(f func(w *World, c *Chunk, dim int32, x0, y0, z0, x1, y1, z1 int)) {
	w.sendLightRegion = f
}
