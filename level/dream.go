package level

import (
	"math"
	"math/rand"
	"sync"

	"github.com/leNicDev/retromc/inventory"
)

const (
	DreamDim   = 2
	DreamFloor = 60

	idWool     = 35
	idChest    = 54
	idSignPost = 63
	idDoor     = 64

	tubeHalf = 7
)

func ClientDim(dim int32) byte {
	if dim == DreamDim {
		return SkyDim
	}
	return byte(dim)
}

type sceneKind int

const (
	sceneTube sceneKind = iota
	scenePlain
	sceneSea
	sceneHall
	sceneIslands
	sceneVoid
)

type palette struct {
	sky, horizon, cloud, grass, dirt, trunk, leaves, sun byte
	stars                                                bool
}

var palettes = []palette{
	{sky: 3, horizon: 0, cloud: 0, grass: 5, dirt: 12, trunk: 12, leaves: 13, sun: 4},
	{sky: 10, horizon: 1, cloud: 6, grass: 13, dirt: 12, trunk: 12, leaves: 13, sun: 14},
	{sky: 15, horizon: 11, cloud: 7, grass: 13, dirt: 12, trunk: 12, leaves: 7, sun: 0, stars: true},
	{sky: 3, horizon: 0, cloud: 0, grass: 13, dirt: 12, trunk: 12, leaves: 1, sun: 4},
	{sky: 8, horizon: 0, cloud: 0, grass: 0, dirt: 12, trunk: 12, leaves: 13, sun: 0},
	{sky: 2, horizon: 6, cloud: 4, grass: 9, dirt: 10, trunk: 15, leaves: 10, sun: 5},
	{sky: 15, horizon: 15, cloud: 7, grass: 8, dirt: 7, trunk: 15, leaves: 15, sun: 14, stars: true},
}

type DreamScene struct {
	kind     sceneKind
	pal      palette
	seed     uint32
	alongZ   bool
	line     int
	oa, ow   int32
	span     int32
	reach    int32
	open     bool
	inverted bool
}

type dreamTile struct {
	x, y, z int32
	sign    *[4]string
	chest   []inventory.Item
}

type dreamState struct {
	mu     sync.Mutex
	scenes map[ChunkCoord]*DreamScene
	root   *DreamScene
}

func (s *DreamScene) au(x, z int) (int, int) {
	if s.alongZ {
		return z, x
	}
	return x, z
}

func (s *DreamScene) chunkAU(cx, cz int32) (int32, int32) {
	if s.alongZ {
		return cz, cx
	}
	return cx, cz
}

func (s *DreamScene) covers(cx, cz int32) bool {
	a, w := s.chunkAU(cx, cz)
	return abs32(a-s.oa) <= s.span && abs32(w-s.ow) <= s.reach
}

func (s *DreamScene) center(a int) int {
	return s.line + int(math.Round(3*math.Sin(float64(a)/14+float64(s.seed%628)/100)))
}

func (s *DreamScene) roll(a, w int, salt int64) float64 {
	return (hash3(int64(a), salt, int64(w), s.seed) + 1) / 2
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func newDreamScene(r *rand.Rand, kind sceneKind, cx, cz int32, alongZ bool, line int) *DreamScene {
	s := &DreamScene{kind: kind, pal: palettes[r.Intn(len(palettes))], seed: r.Uint32(), alongZ: alongZ, line: line, span: 7, reach: 7}
	s.oa, s.ow = s.chunkAU(cx, cz)
	switch kind {
	case sceneTube:
		s.ow = int32(line >> 4)
		s.span, s.reach = 1<<20, 3
		s.inverted = r.Intn(5) == 0
	case sceneVoid:
		s.ow = int32(line >> 4)
		s.span, s.reach = 9, 2
	case sceneHall:
		s.span, s.reach = 5, 5
	}
	return s
}

var sceneWeights = []int{4, 3, 2, 2, 2, 1}

func randomKind(r *rand.Rand) sceneKind {
	total := 0
	for _, w := range sceneWeights {
		total += w
	}
	n := r.Intn(total)
	for k, w := range sceneWeights {
		if n < w {
			return sceneKind(k)
		}
		n -= w
	}
	return sceneTube
}

func (w *World) rootScene() *DreamScene {
	d := &w.dream
	if d.root == nil {
		d.root = newDreamScene(w.dreamRand(), sceneTube, 0, 0, false, 8)
		d.root.pal, d.root.inverted = palettes[0], false
	}
	return d.root
}

func (w *World) dreamRand() *rand.Rand {
	return rand.New(rand.NewSource(rand.Int63()))
}

var dreamNeighbors = []ChunkCoord{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, -1}, {1, -1}, {-1, 1}}

func (w *World) dreamSceneFor(cx, cz int32) *DreamScene {
	d := &w.dream
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.scenes == nil {
		d.scenes = make(map[ChunkCoord]*DreamScene)
	}
	at := ChunkCoord{cx, cz}
	if s, ok := d.scenes[at]; ok {
		return s
	}
	var s *DreamScene
	var from *ChunkCoord
	for _, o := range dreamNeighbors {
		n := ChunkCoord{cx + o.X, cz + o.Z}
		ns, ok := d.scenes[n]
		if !ok {
			continue
		}
		if ns.covers(cx, cz) {
			s = ns
			break
		}
		if from == nil {
			from = &n
		}
	}
	switch {
	case s != nil:
	case from != nil:
		alongZ := from.X == cx
		line := int(cz)*16 + 8
		if alongZ {
			line = int(cx)*16 + 8
		}
		s = newDreamScene(w.dreamRand(), randomKind(w.dreamRand()), cx, cz, alongZ, line)
	default:
		s = w.rootScene()
	}
	d.scenes[at] = s
	return s
}

func (w *World) DreamSceneAt(cx, cz int32) *DreamScene {
	w.dream.mu.Lock()
	defer w.dream.mu.Unlock()
	return w.dream.scenes[ChunkCoord{cx, cz}]
}

func (w *World) DreamShiftScene(px, pz int, alongZ bool, ocx, ocz int32) *DreamScene {
	r := w.dreamRand()
	line := pz
	if alongZ {
		line = px
	}
	if cur := w.DreamSceneAt(int32(px>>4), int32(pz>>4)); cur != nil && cur.kind == sceneTube && r.Intn(5) < 2 {
		s := *cur
		s.open = !cur.open
		if !s.open {
			s.pal = palettes[r.Intn(len(palettes))]
		}
		return &s
	}
	s := newDreamScene(r, randomKind(r), ocx, ocz, alongZ, line)
	if s.kind == sceneTube {
		s.open = r.Intn(4) == 0
	}
	return s
}

func (w *World) DreamSpawn() (float64, float64, float64) {
	s := w.rootScene()
	return 8.5, DreamFloor + 1, float64(s.center(8)) + 0.5
}

func (w *World) DreamChunkCoords() []ChunkCoord {
	coords := make([]ChunkCoord, 0, len(w.dChunks))
	for c := range w.dChunks {
		coords = append(coords, c)
	}
	return coords
}

func (w *World) ResetDream() {
	for c := range w.dChunks {
		w.forgetDreamTiles(c)
	}
	clear(w.dChunks)
	w.dream.mu.Lock()
	w.dream.scenes, w.dream.root = nil, nil
	w.dream.mu.Unlock()
}

func (w *World) AssignDreamScene(coords []ChunkCoord, s *DreamScene) {
	w.dream.mu.Lock()
	defer w.dream.mu.Unlock()
	if w.dream.scenes == nil {
		w.dream.scenes = make(map[ChunkCoord]*DreamScene)
	}
	for _, c := range coords {
		w.dream.scenes[c] = s
	}
}

func (w *World) ReplaceDreamChunk(c ChunkCoord, ch *Chunk) bool {
	if _, loaded := w.dChunks[c]; !loaded || w.DreamSceneAt(c.X, c.Z) != ch.dreamScene {
		return false
	}
	w.forgetDreamTiles(c)
	delete(w.dChunks, c)
	w.InsertChunk(c.X, c.Z, DreamDim, ch)
	return true
}

func (w *World) TrimDream(onRemove func(id int32)) {
	removed := w.PopUnusedChunks(DreamDim)
	if len(removed) == 0 {
		return
	}
	w.CaptureEntities(removed, DreamDim, onRemove)
	w.dream.mu.Lock()
	for c := range removed {
		delete(w.dream.scenes, c)
	}
	if len(w.dChunks) == 0 {
		w.dream.scenes, w.dream.root = nil, nil
	}
	w.dream.mu.Unlock()
	for c := range removed {
		w.forgetDreamTiles(c)
	}
}

func (w *World) forgetDreamTiles(c ChunkCoord) {
	in := func(k BlockKey) bool {
		return k.Dim == DreamDim && WorldToChunkCoord(k.X) == c.X && WorldToChunkCoord(k.Z) == c.Z
	}
	for k := range w.Containers.Chests {
		if in(k) {
			delete(w.Containers.Chests, k)
		}
	}
	for k := range w.Containers.Signs {
		if in(k) {
			delete(w.Containers.Signs, k)
		}
	}
	for k := range w.Containers.Furnaces {
		if in(k) {
			delete(w.Containers.Furnaces, k)
		}
	}
	for k := range w.Containers.Dispensers {
		if in(k) {
			delete(w.Containers.Dispensers, k)
		}
	}
}

func (w *World) adoptDreamTiles(c *Chunk) {
	for _, t := range c.dreamTiles {
		switch {
		case t.chest != nil:
			w.loadChest(t.x, t.y, t.z, DreamDim, t.chest)
		case t.sign != nil:
			w.Containers.Signs[BlockKey{X: t.x, Y: byte(t.y), Z: t.z, Dim: DreamDim}] = *t.sign
		}
	}
	c.dreamTiles = nil
}

type canvas struct {
	x0, z0 int
	blocks []byte
	meta   []byte
	tiles  []dreamTile
	rnd    *rand.Rand
}

func (cv *canvas) in(x, z int) bool {
	return x >= cv.x0 && x < cv.x0+CHUNK_SIZE_X && z >= cv.z0 && z < cv.z0+CHUNK_SIZE_Z
}

func (cv *canvas) set(x, y, z int, id, m byte) {
	if !cv.in(x, z) || y < 0 || y >= CHUNK_SIZE_Y {
		return
	}
	i := blockIndex(x-cv.x0, y, z-cv.z0)
	cv.blocks[i] = id
	setNibble(cv.meta, i, m)
}

func (cv *canvas) get(x, y, z int) byte {
	if !cv.in(x, z) || y < 0 || y >= CHUNK_SIZE_Y {
		return idAir
	}
	return cv.blocks[blockIndex(x-cv.x0, y, z-cv.z0)]
}

func (cv *canvas) wool(x, y, z int, color byte) {
	cv.set(x, y, z, idWool, color)
}

func (cv *canvas) chest(x, y, z int) {
	if !cv.in(x, z) {
		return
	}
	cv.set(x, y, z, idChest, 0)
	cv.tiles = append(cv.tiles, dreamTile{x: int32(x), y: int32(y), z: int32(z), chest: dreamLootChest(cv.rnd)})
}

func (cv *canvas) sign(x, y, z int, rot byte) {
	if !cv.in(x, z) {
		return
	}
	cv.set(x, y, z, idSignPost, rot&15)
	lines := dreamSigns[cv.rnd.Intn(len(dreamSigns))]
	cv.tiles = append(cv.tiles, dreamTile{x: int32(x), y: int32(y), z: int32(z), sign: &lines})
}

func (w *World) generateDreamChunk(cx, cz int32) *Chunk {
	s := w.dreamSceneFor(cx, cz)
	cv := &canvas{
		x0:     int(cx) * CHUNK_SIZE_X,
		z0:     int(cz) * CHUNK_SIZE_Z,
		blocks: make([]byte, chunkBlocksAmount),
		meta:   make([]byte, chunkNibbleCount),
		rnd:    w.dreamRand(),
	}
	switch s.kind {
	case sceneTube:
		s.paintTube(cv)
	case scenePlain:
		s.paintPlain(cv)
	case sceneSea:
		s.paintSea(cv)
	case sceneHall:
		s.paintHall(cv)
	case sceneIslands:
		s.paintIslands(cv)
	case sceneVoid:
		s.paintVoid(cv)
	}
	c := &Chunk{
		X:     cx * CHUNK_SIZE_X,
		Z:     cz * CHUNK_SIZE_Z,
		SizeX: CHUNK_SIZE_X - 1,
		SizeY: CHUNK_SIZE_Y - 1,
		SizeZ: CHUNK_SIZE_Z - 1,
	}
	c.setData(cv.blocks, cv.meta)
	c.needsLight = true
	c.dreamTiles = cv.tiles
	c.dreamScene = s
	return c
}

func (s *DreamScene) columns(cv *canvas, f func(x, z, a, w int)) {
	for lx := range CHUNK_SIZE_X {
		for lz := range CHUNK_SIZE_Z {
			x, z := cv.x0+lx, cv.z0+lz
			a, w := s.au(x, z)
			f(x, z, a, w)
		}
	}
}

func (s *DreamScene) alongRange(cv *canvas, margin int) (int, int) {
	lo := cv.x0
	if s.alongZ {
		lo = cv.z0
	}
	return lo - margin, lo + 15 + margin
}

func (s *DreamScene) xz(a, w int) (int, int) {
	if s.alongZ {
		return w, a
	}
	return a, w
}

func (s *DreamScene) scatter(cv *canvas, cell int, salt int64, chance float64, f func(x, z int)) {
	const margin = 3
	for gx := floorDiv(cv.x0-margin, cell); gx <= floorDiv(cv.x0+15+margin, cell); gx++ {
		for gz := floorDiv(cv.z0-margin, cell); gz <= floorDiv(cv.z0+15+margin, cell); gz++ {
			if s.roll(gx, gz, salt) >= chance {
				continue
			}
			x := gx*cell + int(s.roll(gx, gz, salt+1)*float64(cell))
			z := gz*cell + int(s.roll(gx, gz, salt+2)*float64(cell))
			f(x, z)
		}
	}
}

func floorDiv(a, b int) int {
	return int(math.Floor(float64(a) / float64(b)))
}

func (s *DreamScene) tree(cv *canvas, x, z, base, height int, palm bool, fits func(x, y, z int) bool) {
	for y := base; y < base+height; y++ {
		cv.wool(x, y, z, s.pal.trunk)
	}
	top := base + height
	for dx := -2; dx <= 2; dx++ {
		for dz := -2; dz <= 2; dz++ {
			for dy := -1; dy <= 2; dy++ {
				d := dx*dx + dz*dz + dy*dy*2
				if palm {
					if dy != 0 || (dx != 0 && dz != 0) && absIn(dx)+absIn(dz) > 2 {
						continue
					}
				} else if d > 5 || (dx == 0 && dz == 0 && dy < 0) {
					continue
				}
				lx, ly, lz := x+dx, top+dy, z+dz
				if fits(lx, ly, lz) && cv.get(lx, ly, lz) == idAir {
					cv.wool(lx, ly, lz, s.pal.leaves)
				}
			}
		}
	}
}

func tubeHeight(u int) int {
	f := float64(u) / float64(tubeHalf+1)
	return 5 + int(7*math.Sqrt(1-f*f))
}

func tubeInside(u, v int) bool {
	return v >= 1 && u >= -tubeHalf && u <= tubeHalf && v <= tubeHeight(u)
}

func tubeShell(u, v int) bool {
	if v < 1 || tubeInside(u, v) {
		return false
	}
	return tubeInside(u-1, v) || tubeInside(u+1, v) || tubeInside(u, v-1) || tubeInside(u-1, v-1) || tubeInside(u+1, v-1)
}

func (s *DreamScene) skyBlock(a, u, v int) (byte, byte) {
	if d := mod(a, 48) - 24; absIn(d) <= 1 && absIn(u) <= 1 && v >= 10 {
		if d == 0 && u == 0 {
			return idGlowstone, 0
		}
		return idWool, s.pal.sun
	}
	if absIn(u) == tubeHalf+1 && v == 2 && mod(a, 6) == 0 {
		return idGlowstone, 0
	}
	if (s.pal.stars || s.inverted) && v >= 6 && s.roll(a, u*31+v, 11) > 0.9 {
		return idGlowstone, 0
	}
	if v >= 7 && valueNoise3D(float64(a)/6, float64(u)/3, 0, s.seed+5) > 0.2 {
		return idWool, s.pal.cloud
	}
	if v <= 3 {
		return idWool, s.pal.horizon
	}
	return idWool, s.pal.sky
}

func (s *DreamScene) groundBlock(a, w int, lights float64) (byte, byte) {
	ga, gw := floorDiv(a, 4), floorDiv(w, 4)
	if mod(a, 4) == int(s.roll(ga, gw, 21)*4) && mod(w, 4) == int(s.roll(ga, gw, 22)*4) && s.roll(ga, gw, 23) < lights {
		return idGlowstone, 0
	}
	if r := s.roll(a, w, 24); r > 0.93 {
		return idWool, []byte{14, 4, 0, 2}[int((r-0.93)/0.0175)&3]
	}
	return idWool, s.pal.grass
}

func (s *DreamScene) paintTube(cv *canvas) {
	F := DreamFloor
	lights := 0.75
	if s.pal.stars {
		lights = 0.45
	}
	s.columns(cv, func(x, z, a, w int) {
		u := w - s.center(a)
		if absIn(u) > tubeHalf+1 {
			return
		}
		for v := -3; v < 0; v++ {
			cv.wool(x, F+v, z, s.pal.dirt)
		}
		id, m := s.groundBlock(a, w, lights)
		cv.set(x, F, z, id, m)
		if s.open {
			return
		}
		for v := 1; v <= 14; v++ {
			if tubeShell(u, v) {
				id, m := s.skyBlock(a, u, v)
				cv.set(x, F+v, z, id, m)
			}
		}
	})
	lo, hi := s.alongRange(cv, 3)
	for a := lo; a <= hi; a++ {
		c := s.center(a)
		side := 1
		if s.roll(a, 0, 30) < 0.5 {
			side = -1
		}
		if mod(a, 5) == 0 && s.roll(a, 0, 31) < 0.6 {
			u0 := side * (2 + int(s.roll(a, 0, 32)*4))
			x, z := s.xz(a, c+u0)
			s.tree(cv, x, z, F+1, 3+int(s.roll(a, 0, 33)*2), false, func(x, y, z int) bool {
				ta, tw := s.au(x, z)
				return s.open || tubeInside(tw-s.center(ta), y-F)
			})
		}
		if s.roll(a, 1, 34) < 0.02 {
			x, z := s.xz(a, c+side*(tubeHalf-1))
			cv.chest(x, F+1, z)
		}
		if s.roll(a, 1, 35) < 0.03 {
			x, z := s.xz(a, c-side*3)
			cv.sign(x, F+1, z, s.facing(side < 0))
		}
		if !s.open && !s.inverted && s.roll(a, 1, 36) < 0.012 {
			x, z := s.xz(a, c+side*(tubeHalf+1))
			lower := byte(1)
			if s.alongZ {
				lower = 0
			}
			cv.set(x, F+1, z, idDoor, lower)
			cv.set(x, F+2, z, idDoor, lower|8)
		}
	}
	if s.inverted {
		s.flip(cv, F-3, F+16)
	}
}

func (s *DreamScene) facing(back bool) byte {
	rot := byte(4)
	if s.alongZ {
		rot = 8
	}
	if back {
		rot += 8
	}
	return rot
}

func (s *DreamScene) flip(cv *canvas, lo, hi int) {
	for lx := range CHUNK_SIZE_X {
		for lz := range CHUNK_SIZE_Z {
			for y := lo; y < lo+(hi-lo+1)/2; y++ {
				i, j := blockIndex(lx, y, lz), blockIndex(lx, lo+hi-y, lz)
				cv.blocks[i], cv.blocks[j] = cv.blocks[j], cv.blocks[i]
				mi, mj := getNibble(cv.meta, i), getNibble(cv.meta, j)
				setNibble(cv.meta, i, mj)
				setNibble(cv.meta, j, mi)
			}
		}
	}
	for i := range cv.tiles {
		cv.tiles[i].y = int32(lo + hi - int(cv.tiles[i].y))
	}
}

func (s *DreamScene) paintPlain(cv *canvas) {
	F := DreamFloor
	s.columns(cv, func(x, z, a, w int) {
		for v := -3; v < 0; v++ {
			cv.wool(x, F+v, z, s.pal.dirt)
		}
		_, m := s.groundBlock(a, w, 0)
		cv.wool(x, F, z, m)
		if valueNoise3D(float64(x)/10, 0, float64(z)/10, s.seed+6) > 0.3 {
			cv.wool(x, F+24, z, s.pal.cloud)
			if valueNoise3D(float64(x)/10, 0, float64(z)/10, s.seed+6) > 0.45 {
				cv.wool(x, F+25, z, s.pal.cloud)
			}
		}
	})
	anywhere := func(x, y, z int) bool { return true }
	s.scatter(cv, 7, 40, 0.45, func(x, z int) {
		s.tree(cv, x, z, F+1, 3+int(s.roll(x, z, 41)*3), false, anywhere)
	})
	s.scatter(cv, 23, 42, 0.35, func(x, z int) {
		if cv.get(x, F+1, z) == idAir {
			cv.chest(x, F+1, z)
		}
	})
	s.scatter(cv, 19, 43, 0.4, func(x, z int) {
		if cv.get(x, F+1, z) == idAir {
			cv.sign(x, F+1, z, byte(s.roll(x, z, 44)*16))
		}
	})
}

func (s *DreamScene) paintSea(cv *canvas) {
	F := DreamFloor
	island := func(x, z int) float64 {
		return valueNoise3D(float64(x)/9, 0, float64(z)/9, s.seed+7)
	}
	s.columns(cv, func(x, z, a, w int) {
		cv.wool(x, F-1, z, 11)
		n := island(x, z)
		switch {
		case n > 0.62:
			cv.wool(x, F, z, 4)
			cv.wool(x, F+1, z, 5)
		case n > 0.45:
			cv.wool(x, F, z, 4)
		case mod(a+int(3*math.Sin(float64(w)/4)), 7) == 0:
			cv.wool(x, F, z, 3)
		default:
			cv.wool(x, F, z, 11)
		}
	})
	s.scatter(cv, 9, 50, 0.6, func(x, z int) {
		if island(x, z) > 0.66 {
			s.tree(cv, x, z, F+2, 4, true, func(x, y, z int) bool { return true })
		}
	})
	s.scatter(cv, 13, 51, 0.7, func(x, z int) {
		if n := island(x, z); n > 0.5 && n <= 0.62 && cv.get(x, F+1, z) == idAir {
			if s.roll(x, z, 52) < 0.5 {
				cv.chest(x, F+1, z)
			} else {
				cv.sign(x, F+1, z, byte(s.roll(x, z, 53)*16))
			}
		}
	})
}

func (s *DreamScene) paintHall(cv *canvas) {
	F := DreamFloor
	s.columns(cv, func(x, z, a, w int) {
		cv.wool(x, F-1, z, 7)
		switch {
		case mod(x, 6) == 3 && mod(z, 6) == 3:
			cv.set(x, F, z, idGlowstone, 0)
		case (floorDiv(x, 2)+floorDiv(z, 2))&1 == 0:
			cv.wool(x, F, z, s.pal.horizon)
		default:
			cv.wool(x, F, z, s.pal.sky)
		}
		if mod(x, 9) < 2 && mod(z, 9) < 2 {
			for v := 1; v <= 4; v++ {
				cv.wool(x, F+v, z, s.pal.trunk)
			}
		}
		if mod(x, 3) == 0 && mod(z, 3) == 0 {
			cv.set(x, F+5, z, idGlowstone, 0)
		} else {
			cv.wool(x, F+5, z, s.pal.cloud)
		}
	})
	s.scatter(cv, 17, 60, 0.4, func(x, z int) {
		if cv.get(x, F+1, z) == idAir {
			cv.chest(x, F+1, z)
		}
	})
	s.scatter(cv, 11, 61, 0.35, func(x, z int) {
		if cv.get(x, F+1, z) == idAir {
			cv.sign(x, F+1, z, byte(s.roll(x, z, 62)*16))
		}
	})
}

func (s *DreamScene) paintIslands(cv *canvas) {
	F := DreamFloor
	land := func(x, z int) float64 {
		return valueNoise3D(float64(x)/13, 0, float64(z)/13, s.seed+8)
	}
	s.columns(cv, func(x, z, a, w int) {
		n := land(x, z)
		if n <= 0.1 {
			return
		}
		for v := -1 - int((n-0.1)*14); v < 0; v++ {
			cv.wool(x, F+v, z, s.pal.dirt)
		}
		_, m := s.groundBlock(a, w, 0)
		cv.wool(x, F, z, m)
	})
	s.scatter(cv, 8, 70, 0.5, func(x, z int) {
		if land(x, z) > 0.3 {
			s.tree(cv, x, z, F+1, 3+int(s.roll(x, z, 71)*2), false, func(x, y, z int) bool { return true })
		}
	})
	s.scatter(cv, 21, 72, 0.5, func(x, z int) {
		if land(x, z) > 0.25 && cv.get(x, F+1, z) == idAir {
			cv.chest(x, F+1, z)
		}
	})
}

func (s *DreamScene) paintVoid(cv *canvas) {
	F := DreamFloor
	s.columns(cv, func(x, z, a, w int) {
		if absIn(w-s.center(a)) <= 1 {
			cv.wool(x, F, z, []byte{0, 8}[floorDiv(a, 2)&1])
		}
		for y := F - 24; y <= F+32; y++ {
			if r := s.roll(x*131+y, z, 80); r > 0.9985 {
				cv.wool(x, y, z, byte(r*1e5)&15)
			}
		}
	})
	lo, hi := s.alongRange(cv, 0)
	for a := lo; a <= hi; a++ {
		if s.roll(a, 2, 81) < 0.06 {
			x, z := s.xz(a, s.center(a)+2)
			cv.sign(x, F+1, z, s.facing(s.roll(a, 2, 82) < 0.5))
		}
		if s.roll(a, 2, 83) < 0.01 {
			x, z := s.xz(a, s.center(a))
			cv.chest(x, F+1, z)
		}
	}
}

var dreamSigns = [][4]string{
	{"you have been", "walking for", "a long time", ""},
	{"do not turn", "around", "", ""},
	{"the sky here", "is made of", "wool", ""},
	{"none of this", "is saved", "", ""},
	{"", "WAKE UP", "", ""},
	{"this tube was", "here before", "you were", ""},
	{"chunk", "not found", "", ""},
	{"remember the", "way back?", "", "neither do we"},
	{"fall to", "wake up", "", ""},
	{"the grass is", "soft. it is", "not grass.", ""},
	{"keep walking", "it continues", "as long as", "you do"},
	{"x: ???", "y: ???", "z: ???", ""},
	{"he is not", "in here.", "", "probably."},
	{"there is no", "back. only", "elsewhere.", ""},
	{"the sun is", "a light bulb", "", ""},
	{"you were here", "yesterday", "", "(you weren't)"},
	{"try the door", "", "", ""},
	{"loot ahead", "maybe", "", ""},
	{"everything", "behind you", "is gone", ""},
	{"welcome back", "", "", ""},
	{"seed:", "it changes", "", ""},
	{"hello?", "", "", ""},
	{"this way", "-->", "", ""},
	{"<--", "that way", "", ""},
}

var dreamLoot = []struct {
	id  int16
	max int
}{
	{297, 3}, {260, 4}, {322, 1}, {357, 8}, {354, 1}, {264, 3}, {265, 6}, {266, 5},
	{50, 16}, {35, 32}, {89, 8}, {348, 12}, {345, 1}, {347, 1}, {2256, 1}, {2257, 1},
	{332, 16}, {344, 4}, {341, 3}, {352, 6}, {353, 6}, {321, 2}, {323, 3}, {46, 2},
	{276, 1}, {278, 1}, {261, 1}, {262, 24}, {263, 12},
}

func dreamLootChest(r *rand.Rand) []inventory.Item {
	items := emptyChestItems()
	for range 2 + r.Intn(5) {
		l := dreamLoot[r.Intn(len(dreamLoot))]
		var meta uint16
		if l.id == idWool {
			meta = uint16(r.Intn(16))
		}
		items[r.Intn(len(items))] = inventory.NewItem(l.id, byte(1+r.Intn(l.max)), meta)
	}
	return items
}
