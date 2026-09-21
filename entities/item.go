package entities

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/leNicDev/retromc/constants"
)

const (
	itemWidth  = 0.25
	itemHeight = 0.25

	itemYOffset = 0.125

	itemMaxAge    = 6000
	itemMaxHealth = 5

	itemGravity      = 0.04
	itemVerticalDrag = 0.9800000190734863
	itemWaterPush    = 0.014
	itemPushOutMin   = 0.1
	itemPushOutRange = 0.2

	itemSyncInterval  = 20
	itemFullSyncEvery = 400
	itemSyncThreshold = 1.0 / 32.0
)

// The blocks are only accessed through this, so the physics can be tested without a full world.
type itemWorld interface {
	IsLoaded(x, z, dim int32) bool
	GetBlock(x int32, y byte, z int32, dim int32) constants.WBlock
}

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

	CollectorId int32
	HP          int16

	// Age in ticks, the item is removed after itemMaxAge
	Age int
	// Set once the item is gone (lava, fire, cactus, age, void); the tracker removes it
	Dead     bool
	OnGround bool

	damage int
	inWeb  bool

	// what clients were last told
	lastSyncX, lastSyncY, lastSyncZ float64
	lastFullSync                    int
	settled                         bool
	forceSync                       bool
}

func (d *DroppedItem) GetEntityType() constants.EntityType {
	return constants.DroppedItem
}

func (d *DroppedItem) GetMovementState() *constants.MovementState {
	return &d.MovementState
}

func (d *DroppedItem) Despawn() bool {
	if d.Dead {
		return true
	}
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
	return d.X, d.Y, d.Z
}

func (d *DroppedItem) SetPosition(x, y, z float64) {}

func (d *DroppedItem) GetLoggedIn() bool { return false }

func (d *DroppedItem) GetDim() int32 { return d.Dim }

func (d *DroppedItem) GetVelocity() (float64, float64, float64) { return d.VelX, d.VelY, d.VelZ }

func (d *DroppedItem) InitSyncState() {
	d.lastSyncX, d.lastSyncY, d.lastSyncZ = d.X, d.Y, d.Z
	d.mirrorMovementState()
}

func (d *DroppedItem) mirrorMovementState() {
	ms := &d.MovementState
	ms.X, ms.Y, ms.Z = d.X, d.Y, d.Z
	ms.VelocityX, ms.VelocityY, ms.VelocityZ = d.VelX, d.VelY, d.VelZ
}

func (d *DroppedItem) hurt(amount int) {
	d.damage += amount
	if d.damage >= itemMaxHealth {
		d.Dead = true
	}
}

func (d *DroppedItem) bounds() aabb {
	return aabb{
		minX: d.X - itemWidth/2, minY: d.Y - itemYOffset, minZ: d.Z - itemWidth/2,
		maxX: d.X + itemWidth/2, maxY: d.Y - itemYOffset + itemHeight, maxZ: d.Z + itemWidth/2,
	}
}

func floorInt(v float64) int32 { return int32(math.Floor(v)) }

func blockAt(w itemWorld, x, y, z, dim int32) constants.WBlock {
	if y < 0 || y > 255 || !w.IsLoaded(x, z, dim) {
		return constants.NewAirBlock()
	}
	return w.GetBlock(x, byte(y), z, dim)
}

type vec3 struct{ x, y, z float64 }

func (v vec3) normalized() vec3 {
	lenSq := v.x*v.x + v.y*v.y + v.z*v.z
	if lenSq <= 0 {
		return v
	}
	inv := 1.0 / math.Sqrt(lenSq)
	return vec3{v.x * inv, v.y * inv, v.z * inv}
}

func waterDecay(w itemWorld, x, y, z, dim int32) int {
	b := blockAt(w, x, y, z, dim)
	if !b.IsWater() {
		return -1
	}
	meta := int(b.Metadata)
	if meta >= 8 {
		meta = 0
	}
	return meta
}

var flowDirections = [4][2]int32{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}

func waterFlowVector(w itemWorld, x, y, z, dim int32) vec3 {
	var flow vec3
	own := waterDecay(w, x, y, z, dim)

	for _, dir := range flowDirections {
		nx, nz := x+dir[0], z+dir[1]
		neighbour := waterDecay(w, nx, y, nz, dim)
		if neighbour < 0 {
			if isSolidMaterial(blockAt(w, nx, y, nz, dim)) {
				continue
			}
			below := waterDecay(w, nx, y-1, nz, dim)
			if below < 0 {
				continue
			}
			diff := float64(below - (own - 8))
			flow.x += float64(nx-x) * diff
			flow.z += float64(nz-z) * diff
		} else {
			diff := float64(neighbour - own)
			flow.x += float64(nx-x) * diff
			flow.z += float64(nz-z) * diff
		}
	}

	if b := blockAt(w, x, y, z, dim); b.Metadata >= 8 {
		isWall := func(wx, wy, wz int32) bool {
			nb := blockAt(w, wx, wy, wz, dim)
			return isSolidMaterial(nb) && !nb.IsWater() && nb.TypeId != byte(constants.Ice.Value)
		}
		nearWall := false
		for _, dir := range flowDirections {
			if isWall(x+dir[0], y, z+dir[1]) || isWall(x+dir[0], y+1, z+dir[1]) {
				nearWall = true
				break
			}
		}
		if nearWall {
			flow = flow.normalized()
			flow.y += -6.0
		}
	}

	return flow.normalized()
}

func (d *DroppedItem) handleWater(w itemWorld) bool {
	bb := d.bounds()
	inWater := false
	var push vec3

	for x := floorInt(bb.minX); x <= floorInt(bb.maxX); x++ {
		for y := floorInt(bb.minY); y <= floorInt(bb.maxY); y++ {
			for z := floorInt(bb.minZ); z <= floorInt(bb.maxZ); z++ {
				b := blockAt(w, x, y, z, d.Dim)
				if !b.IsWater() {
					continue
				}
				inWater = true
				flow := waterFlowVector(w, x, y, z, d.Dim)
				push.x += flow.x
				push.y += flow.y
				push.z += flow.z
			}
		}
	}

	push = push.normalized()
	d.VelX += push.x * itemWaterPush
	d.VelY += push.y * itemWaterPush
	d.VelZ += push.z * itemWaterPush
	return inWater
}

func (d *DroppedItem) touchesBlock(w itemWorld, bb aabb, matches func(constants.WBlock) bool) bool {
	for x := floorInt(bb.minX); x <= floorInt(bb.maxX); x++ {
		for y := floorInt(bb.minY); y <= floorInt(bb.maxY); y++ {
			for z := floorInt(bb.minZ); z <= floorInt(bb.maxZ); z++ {
				if matches(blockAt(w, x, y, z, d.Dim)) {
					return true
				}
			}
		}
	}
	return false
}

func isLavaBlock(b constants.WBlock) bool { return b.IsLava() }
func isFireBlock(b constants.WBlock) bool { return b.TypeId == byte(constants.Fire.Value) }

func (d *DroppedItem) pushOutOfBlocks(w itemWorld) {
	bb := d.bounds()
	centerY := (bb.minY + bb.maxY) / 2.0
	bx, by, bz := floorInt(d.X), floorInt(centerY), floorInt(d.Z)
	if !isNormalCube(blockAt(w, bx, by, bz, d.Dim)) {
		return
	}

	fracX := d.X - float64(bx)
	fracY := centerY - float64(by)
	fracZ := d.Z - float64(bz)

	open := func(dx, dy, dz int32) bool {
		return !isNormalCube(blockAt(w, bx+dx, by+dy, bz+dz, d.Dim))
	}

	direction := -1
	closest := 9999.0
	if open(-1, 0, 0) && fracX < closest {
		closest, direction = fracX, 0
	}
	if open(1, 0, 0) && 1.0-fracX < closest {
		closest, direction = 1.0-fracX, 1
	}
	if open(0, -1, 0) && fracY < closest {
		closest, direction = fracY, 2
	}
	if open(0, 1, 0) && 1.0-fracY < closest {
		closest, direction = 1.0-fracY, 3
	}
	if open(0, 0, -1) && fracZ < closest {
		closest, direction = fracZ, 4
	}
	if open(0, 0, 1) && 1.0-fracZ < closest {
		direction = 5
	}

	speed := float64(rand.Float32())*itemPushOutRange + itemPushOutMin
	switch direction {
	case 0:
		d.VelX = -speed
	case 1:
		d.VelX = speed
	case 2:
		d.VelY = -speed
	case 3:
		d.VelY = speed
	case 4:
		d.VelZ = -speed
	case 5:
		d.VelZ = speed
	}
	// the client rolls its own random push speed, so it can't have followed this
	d.forceSync = true
}

func collectCollisionBoxes(w itemWorld, dim int32, area aabb) []aabb {
	var boxes []aabb
	for x := floorInt(area.minX); x <= floorInt(area.maxX); x++ {
		for z := floorInt(area.minZ); z <= floorInt(area.maxZ); z++ {
			if !w.IsLoaded(x, z, dim) {
				continue
			}
			for y := floorInt(area.minY) - 1; y <= floorInt(area.maxY); y++ {
				if y < 0 || y > 255 {
					continue
				}
				for _, local := range blockBoxes(w.GetBlock(x, byte(y), z, dim)) {
					world := local.offset(float64(x), float64(y), float64(z))
					if world.maxX > area.minX && world.minX < area.maxX &&
						world.maxY > area.minY && world.minY < area.maxY &&
						world.maxZ > area.minZ && world.minZ < area.maxZ {
						boxes = append(boxes, world)
					}
				}
			}
		}
	}
	return boxes
}

func (d *DroppedItem) move(w itemWorld) {
	if d.inWeb {
		d.inWeb = false
		// vanilla scales the motion and then zeroes it anyway
		d.VelX, d.VelY, d.VelZ = 0, 0, 0
	}

	origX, origY, origZ := d.VelX, d.VelY, d.VelZ
	dx, dy, dz := origX, origY, origZ

	bb := d.bounds()
	solids := collectCollisionBoxes(w, d.Dim, bb.union(bb.offset(dx, dy, dz)))

	for _, s := range solids {
		dy = clipY(bb, s, dy)
	}
	bb = bb.offset(0, dy, 0)

	for _, s := range solids {
		dx = clipX(bb, s, dx)
	}
	bb = bb.offset(dx, 0, 0)

	for _, s := range solids {
		dz = clipZ(bb, s, dz)
	}
	bb = bb.offset(0, 0, dz)

	d.X = (bb.minX + bb.maxX) / 2
	d.Y = bb.minY + itemYOffset
	d.Z = (bb.minZ + bb.maxZ) / 2

	d.OnGround = dy != origY && origY < 0

	if dx != origX {
		d.VelX = 0
	}
	if dy != origY {
		d.VelY = 0
	}
	if dz != origZ {
		d.VelZ = 0
	}

	d.collideWithBlocks(w, bb)
}

func (d *DroppedItem) collideWithBlocks(w itemWorld, bb aabb) {
	const inset = 0.001
	for x := floorInt(bb.minX + inset); x <= floorInt(bb.maxX-inset); x++ {
		for y := floorInt(bb.minY + inset); y <= floorInt(bb.maxY-inset); y++ {
			for z := floorInt(bb.minZ + inset); z <= floorInt(bb.maxZ-inset); z++ {
				switch blockAt(w, x, y, z, d.Dim).TypeId {
				case byte(constants.Cobweb.Value):
					d.inWeb = true
				case byte(constants.SoulSand.Value):
					d.VelX *= 0.4
					d.VelZ *= 0.4
				case byte(constants.Cactus.Value):
					d.hurt(1)
				}
			}
		}
	}
}

func (d *DroppedItem) groundDrag(w itemWorld) float64 {
	// 0.6f * 0.98f in float32, like the client
	drag := float32(0.58800006)
	below := blockAt(w, floorInt(d.X), floorInt(d.bounds().minY)-1, floorInt(d.Z), d.Dim)
	if below.TypeId != 0 {
		drag = slipperiness(below) * 0.98
	}
	return float64(drag)
}

func (d *DroppedItem) Tick(w itemWorld) {
	if d.Dead {
		return
	}
	// entities in unloaded chunks don't tick
	if !w.IsLoaded(floorInt(d.X), floorInt(d.Z), d.Dim) {
		return
	}

	d.Age++

	d.handleWater(w)
	bb := d.bounds()
	if d.touchesBlock(w, bb, isLavaBlock) {
		// lava deals 4 and burning 1 more per tick, which is more than an item has
		d.Dead = true
		d.mirrorMovementState()
		return
	}
	if d.touchesBlock(w, insetBox(bb, 0.001), isFireBlock) {
		d.hurt(1)
	}
	if d.Y < -64 {
		d.Dead = true
	}
	if d.Dead {
		return
	}

	if d.PickupDelay > 0 {
		d.PickupDelay--
	}

	d.VelY -= itemGravity

	d.pushOutOfBlocks(w)
	d.move(w)
	if d.Dead {
		return
	}

	drag := 0.98
	if d.OnGround {
		drag = d.groundDrag(w)
	}
	d.VelX *= drag
	d.VelY *= itemVerticalDrag
	d.VelZ *= drag

	// bounce, which is always zero because landing already zeroed VelY
	if d.OnGround {
		d.VelY *= -0.5
	}

	if d.Age >= itemMaxAge {
		d.Dead = true
		return
	}

	d.mirrorMovementState()
	d.updateSync()
}

func insetBox(a aabb, by float64) aabb {
	return aabb{a.minX + by, a.minY + by, a.minZ + by, a.maxX - by, a.maxY - by, a.maxZ - by}
}

func (d *DroppedItem) updateSync() {
	hSpeed := math.Hypot(d.VelX, d.VelZ)
	atRest := d.OnGround && hSpeed < 0.005
	if !atRest {
		d.settled = false
	}

	send := d.forceSync
	if atRest && !d.settled {
		d.settled = true
		send = true
	}

	if (d.Age+int(d.EntityId))%itemSyncInterval == 0 {
		moved := math.Max(math.Abs(d.X-d.lastSyncX), math.Max(math.Abs(d.Y-d.lastSyncY), math.Abs(d.Z-d.lastSyncZ)))
		if moved >= itemSyncThreshold {
			send = true
		}
	}
	if d.Age-d.lastFullSync >= itemFullSyncEvery {
		send = true
	}

	if !send {
		return
	}
	d.forceSync = false
	d.lastFullSync = d.Age
	d.lastSyncX, d.lastSyncY, d.lastSyncZ = d.X, d.Y, d.Z
	d.MovementState.Teleported = true
	d.MovementState.VelocityChanged = true
}

func clipY(bb, other aabb, d float64) float64 {
	if other.maxX <= bb.minX || other.minX >= bb.maxX {
		return d
	}
	if other.maxZ <= bb.minZ || other.minZ >= bb.maxZ {
		return d
	}
	if d > 0 && other.minY >= bb.maxY {
		if m := other.minY - bb.maxY; m < d {
			d = m
		}
	} else if d < 0 && other.maxY <= bb.minY {
		if m := other.maxY - bb.minY; m > d {
			d = m
		}
	}
	return d
}

func clipX(bb, other aabb, d float64) float64 {
	if other.maxY <= bb.minY || other.minY >= bb.maxY {
		return d
	}
	if other.maxZ <= bb.minZ || other.minZ >= bb.maxZ {
		return d
	}
	if d > 0 && other.minX >= bb.maxX {
		if m := other.minX - bb.maxX; m < d {
			d = m
		}
	} else if d < 0 && other.maxX <= bb.minX {
		if m := other.maxX - bb.minX; m > d {
			d = m
		}
	}
	return d
}

func clipZ(bb, other aabb, d float64) float64 {
	if other.maxY <= bb.minY || other.minY >= bb.maxY {
		return d
	}
	if other.maxX <= bb.minX || other.minX >= bb.maxX {
		return d
	}
	if d > 0 && other.minZ >= bb.maxZ {
		if m := other.minZ - bb.maxZ; m < d {
			d = m
		}
	} else if d < 0 && other.maxZ <= bb.minZ {
		if m := other.maxZ - bb.minZ; m > d {
			d = m
		}
	}
	return d
}
