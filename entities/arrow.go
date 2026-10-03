package entities

import (
	"math"
	"math/rand"

	c "github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/player"
)

const (
	arrowGravity       = 0.03
	arrowDrag          = 0.99
	arrowWaterDrag     = 0.8
	arrowDamage        = 4
	arrowGroundLife    = 1200
	arrowHalfWidth     = 0.25
	arrowHeight        = 0.5
	arrowHitGrow       = 0.3
	arrowOwnerGrace    = 5
	arrowShakeTicks    = 7
	arrowInaccuracyMul = 0.007499999832361937
)

type Arrow struct {
	EntityId   int32
	OwnerId    int32
	FromPlayer bool
	X, Y, Z    float64
	Vx, Vy, Vz float64
	Yaw, Pitch float32
	Dim        int32

	InGround            bool
	TileX, TileY, TileZ int32
	InTile, InData      byte
	Shake               int32
	CollectorId         int32
	HitId               int32
	Dead                bool

	MovementState c.MovementState

	ticksInAir    int
	ticksInGround int
}

func NewArrow(id int32, owner c.Entity, eyeX, eyeY, eyeZ float64, yaw, pitch float32, dim int32) *Arrow {
	a := &Arrow{EntityId: id, Dim: dim, CollectorId: -1, TileX: -1, TileY: -1, TileZ: -1}
	if owner != nil {
		a.OwnerId = owner.GetEntityId()
		a.FromPlayer = owner.GetEntityType() == c.Player
	}
	yawRad := float64(yaw) / 180 * math.Pi
	pitchRad := float64(pitch) / 180 * math.Pi
	a.X = eyeX - math.Cos(yawRad)*0.16
	a.Y = eyeY - 0.1
	a.Z = eyeZ - math.Sin(yawRad)*0.16
	a.SetHeading(-math.Sin(yawRad)*math.Cos(pitchRad), -math.Sin(pitchRad), math.Cos(yawRad)*math.Cos(pitchRad), 1.5, 1.0)
	return a
}

func (a *Arrow) SetHeading(dx, dy, dz, strength, inaccuracy float64) {
	l := math.Sqrt(dx*dx + dy*dy + dz*dz)
	dx, dy, dz = dx/l, dy/l, dz/l
	dx += rand.NormFloat64() * arrowInaccuracyMul * inaccuracy
	dy += rand.NormFloat64() * arrowInaccuracyMul * inaccuracy
	dz += rand.NormFloat64() * arrowInaccuracyMul * inaccuracy
	a.Vx, a.Vy, a.Vz = dx*strength, dy*strength, dz*strength
	a.faceVelocity()
	a.ticksInGround = 0
	a.syncState(false)
}

func (a *Arrow) faceVelocity() {
	h := math.Sqrt(a.Vx*a.Vx + a.Vz*a.Vz)
	a.Yaw = float32(math.Atan2(a.Vx, a.Vz) * 180 / math.Pi)
	a.Pitch = float32(math.Atan2(a.Vy, h) * 180 / math.Pi)
}

func (a *Arrow) syncState(teleport bool) {
	ms := &a.MovementState
	ms.X, ms.Y, ms.Z = a.X, a.Y, a.Z
	ms.VelocityX, ms.VelocityY, ms.VelocityZ = a.Vx, a.Vy, a.Vz
	ms.Yaw, ms.Pitch = a.Yaw, a.Pitch
	if teleport {
		ms.Teleported = true
	}
}

func (a *Arrow) bounds() aabb {
	return aabb{a.X - arrowHalfWidth, a.Y, a.Z - arrowHalfWidth, a.X + arrowHalfWidth, a.Y + arrowHeight, a.Z + arrowHalfWidth}
}

func (a *Arrow) CanBePickedUpBy(pl *player.Player) bool {
	if !a.InGround || a.Shake > 0 || a.Dead || a.CollectorId != -1 {
		return false
	}
	reach := aabb{pl.X - playerWidth/2 - 1, pl.Y, pl.Z - playerWidth/2 - 1, pl.X + playerWidth/2 + 1, pl.Y + playerHeight, pl.Z + playerWidth/2 + 1}
	return reach.intersects(a.bounds())
}

func (a *Arrow) Tick(w WorldShared) {
	if a.Dead {
		return
	}
	if a.Y < -64 {
		a.Dead = true
		return
	}

	if a.TileY >= 0 {
		tile := blockAt(w, a.TileX, a.TileY, a.TileZ, a.Dim)
		for _, box := range blockBoxes(tile) {
			if box.offset(float64(a.TileX), float64(a.TileY), float64(a.TileZ)).contains(a.X, a.Y, a.Z) {
				a.InGround = true
			}
		}
	}

	if a.Shake > 0 {
		a.Shake--
	}

	if a.InGround {
		tile := blockAt(w, a.TileX, a.TileY, a.TileZ, a.Dim)
		if tile.TypeId == a.InTile && tile.Metadata == a.InData {
			a.ticksInGround++
			if a.ticksInGround >= arrowGroundLife {
				a.Dead = true
			}
			return
		}
		a.InGround = false
		a.Vx *= float64(rand.Float32() * 0.2)
		a.Vy *= float64(rand.Float32() * 0.2)
		a.Vz *= float64(rand.Float32() * 0.2)
		a.ticksInGround = 0
		a.ticksInAir = 0
		a.syncState(true)
	}

	a.ticksInAir++

	sx, sy, sz := a.X, a.Y, a.Z
	ex, ey, ez := sx+a.Vx, sy+a.Vy, sz+a.Vz
	hitBlock, bx, by, bz, frac := a.raycastBlocks(w, sx, sy, sz, ex, ey, ez)
	if hitBlock {
		ex, ey, ez = sx+(ex-sx)*frac, sy+(ey-sy)*frac, sz+(ez-sz)*frac
	}

	if target := a.findEntityHit(w, sx, sy, sz, ex, ey, ez); target != nil {
		var owner c.Entity
		if a.OwnerId != 0 {
			owner, _ = w.GetEntity(a.OwnerId)
		}
		if w.AttackEntity(target, owner, arrowDamage) {
			a.HitId = target.GetEntityId()
			a.Dead = true
			return
		}
		a.Vx, a.Vy, a.Vz = a.Vx*-0.1, a.Vy*-0.1, a.Vz*-0.1
		a.Yaw += 180
		a.ticksInAir = 0
		a.syncState(true)
	} else if hitBlock {
		tile := blockAt(w, bx, by, bz, a.Dim)
		a.TileX, a.TileY, a.TileZ = bx, by, bz
		a.InTile, a.InData = tile.TypeId, tile.Metadata
		a.Vx = float64(float32(ex - a.X))
		a.Vy = float64(float32(ey - a.Y))
		a.Vz = float64(float32(ez - a.Z))
		l := math.Sqrt(a.Vx*a.Vx + a.Vy*a.Vy + a.Vz*a.Vz)
		a.X -= a.Vx / l * 0.05
		a.Y -= a.Vy / l * 0.05
		a.Z -= a.Vz / l * 0.05
		a.InGround = true
		a.Shake = arrowShakeTicks
	}

	a.X += a.Vx
	a.Y += a.Vy
	a.Z += a.Vz

	prevYaw, prevPitch := a.Yaw, a.Pitch
	a.faceVelocity()
	for a.Pitch-prevPitch < -180 {
		prevPitch -= 360
	}
	for a.Pitch-prevPitch >= 180 {
		prevPitch += 360
	}
	for a.Yaw-prevYaw < -180 {
		prevYaw -= 360
	}
	for a.Yaw-prevYaw >= 180 {
		prevYaw += 360
	}
	a.Pitch = prevPitch + (a.Pitch-prevPitch)*0.2
	a.Yaw = prevYaw + (a.Yaw-prevYaw)*0.2

	drag := arrowDrag
	if in := blockAt(w, floorInt(a.X), floorInt(a.Y), floorInt(a.Z), a.Dim); in.IsWater() {
		drag = arrowWaterDrag
	}
	a.Vx *= drag
	a.Vy *= drag
	a.Vz *= drag
	a.Vy -= arrowGravity

	a.syncState(a.InGround)
}

func (a *Arrow) raycastBlocks(w WorldShared, sx, sy, sz, ex, ey, ez float64) (bool, int32, int32, int32, float64) {
	hit := false
	var hx, hy, hz int32
	best := math.Inf(1)
	for x := floorInt(math.Min(sx, ex)); x <= floorInt(math.Max(sx, ex)); x++ {
		for y := floorInt(math.Min(sy, ey)); y <= floorInt(math.Max(sy, ey)); y++ {
			for z := floorInt(math.Min(sz, ez)); z <= floorInt(math.Max(sz, ez)); z++ {
				for _, box := range blockBoxes(blockAt(w, x, y, z, a.Dim)) {
					t, ok := box.offset(float64(x), float64(y), float64(z)).clipSegment(sx, sy, sz, ex, ey, ez)
					if ok && t < best {
						best, hit = t, true
						hx, hy, hz = x, y, z
					}
				}
			}
		}
	}
	return hit, hx, hy, hz, best
}

func (a *Arrow) findEntityHit(w WorldShared, sx, sy, sz, ex, ey, ez float64) c.Entity {
	var target c.Entity
	best := math.Inf(1)
	for _, e := range w.SnapshotEntities() {
		if e.GetDim() != a.Dim || e.GetEntityId() == a.EntityId {
			continue
		}
		if e.GetEntityId() == a.OwnerId && a.ticksInAir < arrowOwnerGrace {
			continue
		}
		var box aabb
		switch v := e.(type) {
		case *Mob:
			if v.HP <= 0 {
				continue
			}
			box = v.bounds()
		case *player.Player:
			if !v.LoggedIn || v.HP <= 0 {
				continue
			}
			box = aabb{v.X - playerWidth/2, v.Y, v.Z - playerWidth/2, v.X + playerWidth/2, v.Y + playerHeight, v.Z + playerWidth/2}
		default:
			continue
		}
		box = aabb{box.minX - arrowHitGrow, box.minY - arrowHitGrow, box.minZ - arrowHitGrow, box.maxX + arrowHitGrow, box.maxY + arrowHitGrow, box.maxZ + arrowHitGrow}
		if t, ok := box.clipSegment(sx, sy, sz, ex, ey, ez); ok && t < best {
			best, target = t, e
		}
	}
	return target
}

func (a *Arrow) GetName() string                          { return "Arrow" }
func (a *Arrow) GetPosition() (float64, float64, float64) { return a.X, a.Y, a.Z }
func (a *Arrow) SetPosition(x, y, z float64)              { a.X, a.Y, a.Z = x, y, z }
func (a *Arrow) GetEntityId() int32                       { return a.EntityId }
func (a *Arrow) SetHP(hp int16)                           {}
func (a *Arrow) GetLoggedIn() bool                        { return false }
func (a *Arrow) GetDim() int32                            { return a.Dim }
func (a *Arrow) GetVelocity() (float64, float64, float64) { return a.Vx, a.Vy, a.Vz }
func (a *Arrow) Despawn() bool                            { return a.Dead }
func (a *Arrow) GetMovementState() *c.MovementState       { return &a.MovementState }
func (a *Arrow) GetEntityType() c.EntityType              { return c.ArrowEntity }

func (a *Arrow) GetHP() int16 {
	if a.Dead {
		return 0
	}
	return 1
}

func (b aabb) contains(x, y, z float64) bool {
	return x > b.minX && x < b.maxX && y > b.minY && y < b.maxY && z > b.minZ && z < b.maxZ
}

func (b aabb) intersects(o aabb) bool {
	return o.maxX > b.minX && o.minX < b.maxX && o.maxY > b.minY && o.minY < b.maxY && o.maxZ > b.minZ && o.minZ < b.maxZ
}

func (b aabb) clipSegment(sx, sy, sz, ex, ey, ez float64) (float64, bool) {
	tMin, tMax := 0.0, 1.0
	axes := [3][4]float64{
		{sx, ex - sx, b.minX, b.maxX},
		{sy, ey - sy, b.minY, b.maxY},
		{sz, ez - sz, b.minZ, b.maxZ},
	}
	for _, ax := range axes {
		start, delta, lo, hi := ax[0], ax[1], ax[2], ax[3]
		if delta == 0 {
			if start < lo || start > hi {
				return 0, false
			}
			continue
		}
		t1, t2 := (lo-start)/delta, (hi-start)/delta
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		tMin, tMax = math.Max(tMin, t1), math.Min(tMax, t2)
		if tMin > tMax {
			return 0, false
		}
	}
	return tMin, true
}
