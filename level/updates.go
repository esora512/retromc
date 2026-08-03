package level

import (
	"log"
	"math"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
)

const (
	MaxFluidSpreadHeight = 7
)

func fluidDelay(b *Block, decay bool) int64 {
	if decay {
		if b.IsWater() {
			return 1
		}
		if b.IsLava() {
			return 25
		}
	} else {
		if b.IsWater() {
			return 15
		}
		if b.IsLava() {
			return 25
		}
	}
	log.Println("WARNING: Using default delay!")
	return 5
}

var neighbours = []struct{ dx, dy, dz int32 }{
	{1, 0, 0}, {-1, 0, 0}, {0, 0, 1}, {0, 0, -1}, {0, 1, 0}, {0, -1, 0},
}

var lateralNeighbors = []struct{ dx, dz int32 }{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
var oppositeLateralNeighbour = []int{1, 0, 3, 2}

// Required because Update trigger is called in package packethandler; cannot import SetBlock due to cycle.
type SetBlock func(x, y, z int32, block Block, dim int32)

type BlockUpdate struct {
	X, Y, Z   int32
	Dimension int32
	SetBlock  SetBlock
}

type BlockUpdateScheduler struct {
	FluidUpdates    map[int64][]BlockUpdate
	FallableUpdates map[int64][]BlockUpdate
}

func NewBlockUpdateScheduler() BlockUpdateScheduler {
	return BlockUpdateScheduler{
		FluidUpdates:    make(map[int64][]BlockUpdate),
		FallableUpdates: make(map[int64][]BlockUpdate),
	}
}

func (s *BlockUpdateScheduler) applyFluidUpdates(tick int64) []BlockUpdate {
	updates := s.FluidUpdates[tick]
	delete(s.FluidUpdates, tick)
	return updates
}

func (s *BlockUpdateScheduler) applyFallableUpdates(tick int64) []BlockUpdate {
	updates := s.FallableUpdates[tick]
	delete(s.FallableUpdates, tick)
	return updates
}

func (s *BlockUpdateScheduler) scheduleFluidUpdate(tick int64, x, y, z int32, setBlock SetBlock, dim int32) {
	s.FluidUpdates[tick] = append(s.FluidUpdates[tick], BlockUpdate{X: x, Y: y, Z: z, SetBlock: setBlock, Dimension: dim})
}

func (s *BlockUpdateScheduler) scheduleFallableUpdate(tick int64, x, y, z int32, setBlock SetBlock, dim int32) {
	for _, job := range s.FallableUpdates[tick] {
		if job.X == x && job.Y == y && job.Z == z {
			return // already scheduled for this tick
		}
	}
	s.FallableUpdates[tick] = append(s.FallableUpdates[tick], BlockUpdate{X: x, Y: y, Z: z, SetBlock: setBlock, Dimension: dim})
}

func notifyFluid(w *World, x, y, z int32, setBlock SetBlock, dim int32) {
	if y < 0 || y > 255 || !w.IsLoaded(x, z, dim) {
		return
	}
	b := w.GetBlock(x, byte(y), z, dim)
	if !b.IsFluid() {
		return
	}
	delay := fluidDelay(&b, true)
	w.Scheduler.scheduleFluidUpdate(w.Tick+delay, x, y, z, setBlock, dim)
}

func notifyFallable(w *World, x, y, z int32, setBlock SetBlock, dim int32) {
	if y < 0 || y > 255 || !w.IsLoaded(x, z, dim) {
		return
	}
	b := w.GetBlock(x, byte(y), z, dim)
	if b.TypeId != byte(constants.Sand.Value) && b.TypeId != byte(constants.Gravel.Value) {
		return
	}
	w.Scheduler.scheduleFallableUpdate(w.Tick+5, x, y, z, setBlock, dim)

}

func notifyFallableNeighbours(w *World, x, y, z int32, setBlock SetBlock, dim int32) {
	for _, n := range neighbours {
		notifyFallable(w, x+n.dx, y+n.dy, z+n.dz, setBlock, dim)
	}
}

func notifyFluidNeighbors(w *World, x, y, z int32, setBlock SetBlock, dim int32) {
	for _, n := range neighbours {
		notifyFluid(w, x+n.dx, y+n.dy, z+n.dz, setBlock, dim)
	}
}

func processFluidUpdate(w *World, u *BlockUpdate) {
	b := w.GetBlock(u.X, byte(u.Y), u.Z, u.Dimension)
	if !b.IsFluid() {
		return
	}

	if b.IsLava() {
		log.Printf("x=%d, y=%d, z=%d is lava", u.X, u.Y, u.Z)
		if tryHardenLava(w, u.X, u.Y, u.Z, b, u.SetBlock, u.Dimension) {
			return
		}
	}

	recomputeFluid(w, u, b)
}

func processFallableUpdateJob(w *World, u *BlockUpdate) {
	b := w.GetBlock(u.X, byte(u.Y), u.Z, u.Dimension)
	if b.TypeId != byte(constants.Sand.Value) && b.TypeId != byte(constants.Gravel.Value) {
		return
	}
	beneath := w.GetBlock(u.X, byte(u.Y)-1, u.Z, u.Dimension)
	if !beneath.IsAir() && !beneath.IsLiquid() {
		return
	}

	air := NewAirBlock()
	u.SetBlock(u.X, u.Y, u.Z, air, u.Dimension)

	if !w.areaLoaded(u.X, u.Z, 32, u.Dimension) {
		w.instaFallAt(u.X, u.Z, u.Y, int16(b.TypeId), byte(b.Metadata), u.Dimension)
		notifyFallableNeighbours(w, u.X, u.Y, u.Z, u.SetBlock, u.Dimension)
		return
	}

	objectType := byte(0)
	if b.TypeId == byte(constants.Sand.Value) {
		objectType = 70
	} else if b.TypeId == byte(constants.Gravel.Value) {
		objectType = 71
	}

	entityId := w.NextEntityId()
	falling := entities.NewBlockEntity(entityId, int16(b.TypeId), byte(b.Metadata), float64(u.X), float64(u.Y), float64(u.Z), u.Dimension)
	w.BroadcastSpawnObject(entityId, objectType, int32(math.Floor((float64(u.X)+0.5)*32)), int32(math.Floor(float64(u.Y)*32)), int32(math.Floor((float64(u.Z)+0.5)*32)), 0, 0, 0, 0)
	if falling.Dimension == -1 {
		log.Println("Adding Falling Entity in Nether")
	}
	w.AddEntity(falling)
	notifyFallableNeighbours(w, falling.X, int32(falling.Y), falling.Z, u.SetBlock, u.Dimension)
}

func (w *World) TickFluids() {
	updates := w.Scheduler.applyFluidUpdates(w.Tick)
	for _, update := range updates {
		processFluidUpdate(w, &update)
	}
}

func (w *World) TickFallables() {
	updates := w.Scheduler.applyFallableUpdates(w.Tick)
	for _, update := range updates {
		processFallableUpdateJob(w, &update)
	}
}

func (w *World) TriggerFluidUpdate(x, y, z int32, setBlock SetBlock, dim int32) {
	notifyFluid(w, x, y, z, setBlock, dim)
	notifyFluidNeighbors(w, x, y, z, setBlock, dim)
}

func (w *World) TriggerFallableUpdate(x, y, z int32, setBlock SetBlock, dim int32) {
	notifyFallable(w, x, y, z, setBlock, dim)
	notifyFallableNeighbours(w, x, y, z, setBlock, dim)
}

func recomputeFluid(w *World, u *BlockUpdate, b Block) {
	isSource := b.IsStillWater() || b.IsStillLava()

	if !isSource && b.IsWater() {
		if hasSolidSupport(w, u.X, u.Y, u.Z, u.Dimension) && countAdjacentWaterSources(w, u.X, u.Y, u.Z, u.Dimension) >= 2 {
			source := NewStillWaterBlock(0)
			u.SetBlock(u.X, u.Y, u.Z, source, u.Dimension)
			notifyFluidNeighbors(w, u.X, u.Y, u.Z, u.SetBlock, u.Dimension)
			return
		}
	}

	if !isSource {
		newLevel, hasSupport := idealFluidLevel(w, u.X, u.Y, u.Z, u.Dimension)
		if !hasSupport {
			air := NewAirBlock()
			u.SetBlock(u.X, u.Y, u.Z, air, u.Dimension)
			notifyFluidNeighbors(w, u.X, u.Y, u.Z, u.SetBlock, u.Dimension)
			return
		}
		if newLevel != int(b.Metadata) {
			var updated Block
			if b.IsWater() {
				updated = NewFlowingWaterBlock(byte(newLevel))
			}
			if b.IsLava() {
				updated = NewFlowingLavaBlock(byte(newLevel))
			}
			u.SetBlock(u.X, u.Y, u.Z, updated, u.Dimension)
			b = updated
			notifyFluidNeighbors(w, u.X, u.Y, u.Z, u.SetBlock, u.Dimension)
		}
	}
	trySpread(w, u.X, u.Y, u.Z, b, u.SetBlock, u.Dimension)
}

func idealFluidLevel(w *World, x, y, z int32, dim int32) (int, bool) {
	// Determines fluid height of next fluid block
	if y < 255 {
		above := w.GetBlock(x, byte(y+1), z, dim)
		if above.IsFluid() {
			return 0, true
		}
	}
	best := -1
	for _, n := range lateralNeighbors {
		nx, nz := x+n.dx, z+n.dz
		if !w.IsLoaded(nx, nz, dim) {
			continue
		}
		nb := w.GetBlock(nx, byte(y), nz, dim)
		if !nb.IsFluid() {
			continue
		}

		var contribution int
		if nb.IsStillWater() || nb.IsStillLava() {
			contribution = 1
		} else {
			contribution = int(nb.Metadata) + 1
		}
		if contribution > MaxFluidSpreadHeight {
			continue
		}
		if best == -1 || contribution < best {
			best = contribution
		}
	}

	if best == -1 {
		return 0, false
	}
	return best, true
}

func trySpread(w *World, x, y, z int32, b Block, setBlock SetBlock, dim int32) {
	isSource := b.IsStillWater() || b.IsStillLava()
	level := 0
	if !isSource {
		level = int(b.Metadata)
	}

	var flowing Block
	if b.IsWater() {
		flowing = NewFlowingWaterBlock(0)
	}
	if b.IsLava() {
		flowing = NewFlowingLavaBlock(0)
	}
	fedBelow := trySpreadInto(w, x, y-1, z, flowing, setBlock, dim)

	belowSolid := true
	if y > 0 {
		below := w.GetBlock(x, byte(y-1), z, dim)
		belowSolid = !below.IsAir() && !below.IsFluid()
	}

	if !isSource && fedBelow {
		return
	}
	if !isSource && !belowSolid {
		return
	}

	newLevel := level + 1
	if newLevel > MaxFluidSpreadHeight {
		return
	}

	for _, d := range spreadDirections(w, x, y, z, dim) {
		var flowing Block
		if b.IsWater() {
			flowing = NewFlowingWaterBlock(byte(newLevel))
		}
		if b.IsLava() {
			flowing = NewFlowingLavaBlock(byte(newLevel))
		}
		trySpreadInto(w, x+d.dx, y, z+d.dz, flowing, setBlock, dim)
	}
}

func spreadDirections(w *World, x, y, z int32, dim int32) []struct{ dx, dz int32 } {
	const maxDepth = 4
	const unreachable = 1 << 30

	dist := make([]int, len(lateralNeighbors))
	best := unreachable

	for i, n := range lateralNeighbors {
		nx, nz := x+n.dx, z+n.dz
		dist[i] = unreachable

		if isBlockedForFlow(w, nx, y, nz, dim) {
			continue
		}
		if isOpenBelow(w, nx, y, nz, dim) {
			dist[i] = 0
		} else {
			dist[i] = distanceToGap(w, nx, y, nz, 1, i, maxDepth, dim)
		}

		if dist[i] < best {
			best = dist[i]
		}
	}

	if best == unreachable {
		return lateralNeighbors // no gap nearby -- ordinary unrestricted spread
	}

	var dirs []struct{ dx, dz int32 }
	for i, n := range lateralNeighbors {
		if dist[i] == best {
			dirs = append(dirs, n)
		}
	}
	return dirs
}

func distanceToGap(w *World, x, y, z int32, dist, fromDir int, maxDepth int, dim int32) int {
	const unreachable = 1 << 30
	minDist := unreachable

	for i, n := range lateralNeighbors {
		if i == oppositeLateralNeighbour[fromDir] {
			continue
		}

		nx, nz := x+n.dx, z+n.dz
		if isBlockedForFlow(w, nx, y, nz, dim) {
			continue
		}
		if isOpenBelow(w, nx, y, nz, dim) {
			return dist
		}
		if dist >= maxDepth {
			continue
		}

		d := distanceToGap(w, nx, y, nz, dist+1, i, maxDepth, dim)
		if d < minDist {
			minDist = d
		}
	}

	return minDist
}

func isBlockedForFlow(w *World, x, y, z int32, dim int32) bool {
	if !w.IsLoaded(x, z, dim) {
		return true
	}
	b := w.GetBlock(x, byte(y), z, dim)
	if isFluidReplaceable(b) || b.IsFluid() {
		return false
	}
	return true
}

func isOpenBelow(w *World, x, y, z int32, dim int32) bool {
	if y <= 0 {
		return false
	}
	b := w.GetBlock(x, byte(y-1), z, dim)
	return b.IsAir()
}

func trySpreadInto(w *World, x, y, z int32, flowing Block, setBlock SetBlock, dim int32) bool {
	if y < 0 || y > 255 || !w.IsLoaded(x, z, dim) {
		return false
	}
	b := w.GetBlock(x, byte(y), z, dim)
	if !isFluidReplaceable(b) {
		return false
	}

	setBlock(x, y, z, flowing, dim)
	delay := fluidDelay(&flowing, false)
	w.Scheduler.scheduleFluidUpdate(w.Tick+delay, x, y, z, setBlock, dim)
	return true
}

var hardenNeighbors = []struct{ dx, dy, dz int32 }{
	{1, 0, 0}, {-1, 0, 0}, {0, 0, 1}, {0, 0, -1}, {0, 1, 0},
}

func lavaTouchesWater(w *World, x, y, z int32, dim int32) bool {
	for _, n := range hardenNeighbors {
		nx := x + n.dx
		ny := y + n.dy
		nz := z + n.dz

		if ny < 0 || ny > 255 {
			continue
		}
		if !w.IsLoaded(nx, nz, dim) {
			continue
		}

		b := w.GetBlock(nx, byte(ny), nz, dim)
		if b.IsWater() {
			return true
		}
	}
	return false
}

func tryHardenLava(w *World, x, y, z int32, b Block, setBlock SetBlock, dim int32) bool {
	if !lavaTouchesWater(w, x, y, z, dim) {
		return false
	}

	if b.IsStillLava() {
		obsidian := NewObsidianBlock()
		setBlock(x, y, z, obsidian, dim)
	} else {
		if b.Metadata > 4 {
			return false
		}
		cobble := NewCobblestoneBlock()
		setBlock(x, y, z, cobble, dim)
	}

	notifyFluidNeighbors(w, x, y, z, setBlock, dim)
	return true
}

func countAdjacentWaterSources(w *World, x, y, z int32, dim int32) int {
	count := 0
	for _, n := range lateralNeighbors {
		nx, nz := x+n.dx, z+n.dz
		if !w.IsLoaded(nx, nz, dim) {
			continue
		}
		nb := w.GetBlock(nx, byte(y), nz, dim)
		if nb.IsStillWater() {
			count++
		}
	}
	return count
}

func hasSolidSupport(w *World, x, y, z int32, dim int32) bool {
	if y <= 0 {
		return true
	}
	below := w.GetBlock(x, byte(y-1), z, dim)
	return !below.IsAir() && !below.IsFluid()
}

func isFluidReplaceable(b Block) bool {
	return b.IsAir() ||
		b.IsSnowLayer() ||
		b.TypeId == byte(constants.Fire.Value) ||
		b.TypeId == byte(constants.Dandelion.Value) ||
		b.TypeId == byte(constants.Rose.Value) ||
		b.TypeId == byte(constants.BrownMushroom.Value) ||
		b.TypeId == byte(constants.RedMushroom.Value) ||
		b.TypeId == byte(constants.Tallgrass.Value)
}
