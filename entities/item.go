package entities

import (
	"fmt"
	"math"

	"github.com/leNicDev/retromc/constants"
)

type DroppedItem struct {
	EntityId    int32
	ItemId      int32
	Amount      byte
	Metadata    byte
	X, Y, Z     float64
	PickupDelay int32
	Dim         int32

	VelX, VelY, VelZ float64

	DespawnIn     int
	MovementState constants.MovementState
	InLava        bool

	CollectorId int32
	HP          int16
}

func (d *DroppedItem) GetEntityType() constants.EntityType {
	return constants.DroppedItem
}

func (d *DroppedItem) GetMovementState() *constants.MovementState {
	return &d.MovementState
}

func (d *DroppedItem) Despawn() bool {
	if d.DespawnIn < 0 {
		return false
	}
	if d.DespawnIn == 0 {
		d.DespawnIn = -1
		return true
	}
	d.DespawnIn -= 1
	return false
}

func (d *DroppedItem) GetEntityId() int32 {
	return d.EntityId
}

func (d *DroppedItem) GetHP() int16 {
	return d.HP
}

func (d *DroppedItem) SetHP(hp int16) {
	d.HP = hp
}

func (d *DroppedItem) GetName() string {
	return fmt.Sprintf("Entity %d", d.EntityId)
}

func (d *DroppedItem) GetPosition() (float64, float64, float64) {
	return float64(d.X), float64(d.Y), float64(d.Z)
}

func (d *DroppedItem) SetPosition(x, y, z float64) {}

func (d *DroppedItem) GetLoggedIn() bool { return false }

func (d *DroppedItem) GetDim() int32 { return d.Dim }

func (d *DroppedItem) GetVelocity() (float64, float64, float64) { return d.VelX, d.VelY, d.VelZ }

// Vibed dropped item fluid handling because can't be bothered with math and it works, lel
const itemColliderHalfWidth = 0.125
const itemColliderHeight = 0.25

func itemAABB(d *DroppedItem) (minX, minY, minZ, maxX, maxY, maxZ float64) {
	return d.X - itemColliderHalfWidth, d.Y, d.Z - itemColliderHalfWidth,
		d.X + itemColliderHalfWidth, d.Y + itemColliderHeight, d.Z + itemColliderHalfWidth
}

func fluidHeight(b constants.WBlock) float64 {
	if b.IsStillWater() || b.IsStillLava() {
		return 1.0
	}
	level := int(b.Metadata)
	if level > 7 {
		level = 7
	}
	return float64(7-level) / 8.0
}

type FlowVec struct{ X, Z float64 }

var lateralOffsets = []struct{ dx, dz int32 }{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

const (
	gravity           = 0.04
	airDrag           = 0.98
	groundDragBase    = 0.58800006
	waterFlowStrength = 0.014
	buoyancy          = 0.02
	buoyancyMaxUp     = 0.06
)

func getFlowVector(w WorldShared, bx int32, by byte, bz int32, dim int32, b constants.WBlock) FlowVec {
	isLava := b.IsLava()
	ownHeight := fluidHeight(b)

	var flow FlowVec
	for _, n := range lateralOffsets {
		nx, nz := bx+n.dx, bz+n.dz
		if !w.IsLoaded(nx, nz, dim) {
			continue
		}
		nb := w.GetBlock(nx, by, nz, dim)

		var nHeight float64
		switch {
		case isLava && nb.IsLava():
			nHeight = fluidHeight(nb)
		case !isLava && nb.IsWater():
			nHeight = fluidHeight(nb)
		case nb.IsFluidReplaceable():
			// air/flowers etc: if fluid continues one block down, treat this
			// direction as a steep drop so flow pulls toward the edge/fall
			if by == 0 {
				continue
			}
			below := w.GetBlock(nx, by-1, nz, dim)
			if (isLava && below.IsLava()) || (!isLava && below.IsWater()) {
				nHeight = ownHeight - 1.0
			} else {
				continue
			}
		default:
			continue
		}

		diff := ownHeight - nHeight
		flow.X += float64(n.dx) * diff
		flow.Z += float64(n.dz) * diff
	}

	if length := math.Hypot(flow.X, flow.Z); length > 1e-4 {
		flow.X /= length
		flow.Z /= length
	}
	return flow
}

func handleFluidAcceleration(d *DroppedItem, w WorldShared) bool {
	minX, minY, minZ, maxX, maxY, maxZ := itemAABB(d)

	bx0, bx1 := int32(math.Floor(minX)), int32(math.Floor(maxX))
	by0, by1 := int32(math.Floor(minY)), int32(math.Floor(maxY))
	bz0, bz1 := int32(math.Floor(minZ)), int32(math.Floor(maxZ))

	touched := false
	var accumX, accumZ float64
	var accumY float64

	for bx := bx0; bx <= bx1; bx++ {
		for by := by0; by <= by1; by++ {
			if by < 0 || by > 255 {
				continue
			}
			for bz := bz0; bz <= bz1; bz++ {
				if !w.IsLoaded(bx, bz, d.Dim) {
					continue
				}
				b := w.GetBlock(bx, byte(by), bz, d.Dim)
				if !b.IsWater() {
					continue
				}

				h := fluidHeight(b)
				blockTop := float64(by) + h
				if blockTop < minY {
					continue
				}
				touched = true

				flow := getFlowVector(w, bx, byte(by), bz, d.Dim, b)
				accumX += flow.X
				accumZ += flow.Z
				accumY += 1.0
			}
		}
	}

	if !touched {
		return false
	}

	if length := math.Hypot(accumX, accumZ); length > 1e-4 {
		accumX /= length
		accumZ /= length
	}

	d.VelX += accumX * waterFlowStrength
	d.VelZ += accumZ * waterFlowStrength
	if accumY > 0 && d.VelY < buoyancyMaxUp {
		d.VelY += buoyancy
	}
	return true
}

func touchingLava(d *DroppedItem, w WorldShared) bool {
	minX, minY, minZ, maxX, maxY, maxZ := itemAABB(d)

	bx0, bx1 := int32(math.Floor(minX)), int32(math.Floor(maxX))
	by0, by1 := int32(math.Floor(minY)), int32(math.Floor(maxY))
	bz0, bz1 := int32(math.Floor(minZ)), int32(math.Floor(maxZ))

	for bx := bx0; bx <= bx1; bx++ {
		for by := by0; by <= by1; by++ {
			if by < 0 || by > 255 {
				continue
			}
			for bz := bz0; bz <= bz1; bz++ {
				if !w.IsLoaded(bx, bz, d.Dim) {
					continue
				}
				b := w.GetBlock(bx, byte(by), bz, d.Dim)
				if !b.IsLava() {
					continue
				}
				h := fluidHeight(b)
				if float64(by)+h >= minY {
					return true
				}
			}
		}
	}
	return false
}

func (d *DroppedItem) Tick(w WorldShared) {
	if d.InLava {
		return
	}

	if touchingLava(d, w) {
		d.DespawnIn = 3
		d.VelX = 0
		d.VelY = 0
		d.VelZ = 0
		d.MovementState.VelocityX = 0
		d.MovementState.VelocityY = 0
		d.MovementState.VelocityZ = 0
		d.InLava = true
		return
	}

	if d.PickupDelay > 0 {
		d.PickupDelay--
	}

	handleFluidAcceleration(d, w)

	d.VelY -= gravity

	box := boundingBoxAt(d.X, d.Y, d.Z)
	solids := collectSolidBoxes(w, d.Dim, box.union(box.offset(d.VelX, d.VelY, d.VelZ)))

	origVelY := d.VelY
	dx, dy, dz := d.VelX, d.VelY, d.VelZ

	for _, s := range solids {
		dy = clipY(box, s, dy)
	}
	box = box.offset(0, dy, 0)

	for _, s := range solids {
		dx = clipX(box, s, dx)
	}
	box = box.offset(dx, 0, 0)

	for _, s := range solids {
		dz = clipZ(box, s, dz)
	}
	box = box.offset(0, 0, dz)

	onGround := dy != origVelY && origVelY < 0

	d.X = (box.minX + box.maxX) / 2
	d.Y = box.minY
	d.Z = (box.minZ + box.maxZ) / 2

	if dx != d.VelX {
		d.VelX = 0
	}
	if dz != d.VelZ {
		d.VelZ = 0
	}
	if dy != d.VelY {
		d.VelY = 0
	}

	drag := float64(airDrag)
	if onGround {
		drag = groundDragBase
	}

	d.VelX *= drag
	d.VelZ *= drag
	d.VelY *= 0.9800000190734863
}

type aabb struct {
	minX, minY, minZ float64
	maxX, maxY, maxZ float64
}

func boundingBoxAt(x, y, z float64) aabb {
	return aabb{
		minX: x - itemColliderHalfWidth, minY: y, minZ: z - itemColliderHalfWidth,
		maxX: x + itemColliderHalfWidth, maxY: y + itemColliderHeight, maxZ: z + itemColliderHalfWidth,
	}
}

func (a aabb) union(o aabb) aabb {
	return aabb{
		minX: math.Min(a.minX, o.minX), minY: math.Min(a.minY, o.minY), minZ: math.Min(a.minZ, o.minZ),
		maxX: math.Max(a.maxX, o.maxX), maxY: math.Max(a.maxY, o.maxY), maxZ: math.Max(a.maxZ, o.maxZ),
	}
}

func (a aabb) offset(dx, dy, dz float64) aabb {
	a.minX += dx
	a.maxX += dx
	a.minY += dy
	a.maxY += dy
	a.minZ += dz
	a.maxZ += dz
	return a
}

// blocks the item could possibly touch this tick: current box unioned with
// where it wants to move to. This is what fixes the corner cases — you're
// querying every block the swept box could clip against, not one point.
func collectSolidBoxes(w WorldShared, dim int32, box aabb) []aabb {
	bx0, bx1 := int32(math.Floor(box.minX)), int32(math.Floor(box.maxX))
	by0, by1 := int32(math.Floor(box.minY)), int32(math.Floor(box.maxY))
	bz0, bz1 := int32(math.Floor(box.minZ)), int32(math.Floor(box.maxZ))

	var boxes []aabb
	for bx := bx0; bx <= bx1; bx++ {
		for by := by0; by <= by1; by++ {
			if by < 0 || by > 255 {
				continue
			}
			for bz := bz0; bz <= bz1; bz++ {
				if !w.IsLoaded(bx, bz, dim) {
					continue
				}
				b := w.GetBlock(bx, byte(by), bz, dim)
				if b.IsLiquid() || !b.IsSolid() {
					continue
				}
				boxes = append(boxes, aabb{
					minX: float64(bx), minY: float64(by), minZ: float64(bz),
					maxX: float64(bx) + 1, maxY: float64(by) + 1, maxZ: float64(bz) + 1,
				})
			}
		}
	}
	return boxes
}

// clip a proposed movement `d` along one axis so the moving box doesn't
// pass through `other`. Mirrors AxisAlignedBB.calculateXOffset/Y/Z in vanilla.
func clipY(box, other aabb, d float64) float64 {
	if other.maxX <= box.minX || other.minX >= box.maxX {
		return d
	}
	if other.maxZ <= box.minZ || other.minZ >= box.maxZ {
		return d
	}
	if d > 0 && other.minY >= box.maxY {
		if m := other.minY - box.maxY; m < d {
			d = m
		}
	} else if d < 0 && other.maxY <= box.minY {
		if m := other.maxY - box.minY; m > d {
			d = m
		}
	}
	return d
}

func clipX(box, other aabb, d float64) float64 {
	if other.maxY <= box.minY || other.minY >= box.maxY {
		return d
	}
	if other.maxZ <= box.minZ || other.minZ >= box.maxZ {
		return d
	}
	if d > 0 && other.minX >= box.maxX {
		if m := other.minX - box.maxX; m < d {
			d = m
		}
	} else if d < 0 && other.maxX <= box.minX {
		if m := other.maxX - box.minX; m > d {
			d = m
		}
	}
	return d
}

func clipZ(box, other aabb, d float64) float64 {
	if other.maxY <= box.minY || other.minY >= box.maxY {
		return d
	}
	if other.maxX <= box.minX || other.minX >= box.maxX {
		return d
	}
	if d > 0 && other.minZ >= box.maxZ {
		if m := other.minZ - box.maxZ; m < d {
			d = m
		}
	} else if d < 0 && other.maxZ <= box.minZ {
		if m := other.maxZ - box.minZ; m > d {
			d = m
		}
	}
	return d
}
