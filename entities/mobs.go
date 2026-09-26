package entities

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/rand"

	"github.com/leNicDev/retromc/constants"
	c "github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/player"
)

const (
	mobGravity     = 0.08
	mobJumpVel     = 0.42
	mobStepHeight  = 0.5
	mobMaxHurtTime = 20
	mobMaxAir      = 300
	mobLookRange   = 8.0
	mobAggroRange  = 16.0
	playerHeight   = 1.8
	playerWidth    = 0.6
)

type Mob struct {
	EntityId   int32
	X, Y, Z    float64
	Vx, Vy, Vz float64
	Yaw        byte
	Pitch      byte
	HP         int16
	Dimension  int32

	TargetId int32
	MobType  byte
	Metadata byte

	OnGround       bool
	AttackCooldown int32

	ShouldDespawn bool
	DespawnIn     int

	MovementState c.MovementState

	RotYaw, RotPitch float32
	Age              int32
	FireTicks        int32
	Color            byte
	Sheared          bool

	forward, strafe float32
	jumping         bool
	collidedH       bool
	inWater         bool
	inWeb           bool
	fallDistance    float64
	hurtTime        int32
	lastDamage      int16
	air             int32

	hasGoal             bool
	goalX, goalY, goalZ int32
	goalTicks           int32

	lookTarget   int32
	lookTicks    int32
	randomYawVel float32

	hasAttacked  bool
	fuse         int32
	creeperState int8
	metaDirty    bool
}

func NewMob(id int32, mobType byte, x, y, z float64, dim int32) *Mob {
	m := &Mob{
		EntityId:     id,
		X:            x,
		Y:            y,
		Z:            z,
		Dimension:    dim,
		MobType:      mobType,
		TargetId:     -1,
		lookTarget:   -1,
		HP:           MaxHealth(mobType),
		OnGround:     true,
		DespawnIn:    -1,
		air:          mobMaxAir,
		creeperState: -1,
		RotYaw:       rand.Float32() * 360,
	}
	if mobType == c.Sheep {
		m.Color = rollFleeceColor()
	}
	m.syncState()
	return m
}

func rollFleeceColor() byte {
	roll := rand.Intn(100)
	switch {
	case roll < 5:
		return 15
	case roll < 10:
		return 7
	case roll < 15:
		return 8
	case roll < 18:
		return 12
	case rand.Intn(500) == 0:
		return 6
	}
	return 0
}

func MaxHealth(mobType byte) int16 {
	switch mobType {
	case c.Pig:
		return 10
	case c.Sheep:
		return 8
	}
	return 20
}

func (m *Mob) size() (width, height float64) {
	switch m.MobType {
	case c.Spider:
		return 1.4, 0.9
	case c.Pig:
		return 0.9, 0.9
	case c.Sheep:
		return 0.9, 1.3
	}
	return 0.6, 1.8
}

func (m *Mob) eyeHeight() float64 {
	_, h := m.size()
	return h * 0.85
}

func (m *Mob) speed() float32 {
	switch m.MobType {
	case c.Zombie:
		return 0.5
	case c.Spider:
		return 0.8
	}
	return 0.7
}

func (m *Mob) attackStrength() int16 {
	if m.MobType == c.Zombie {
		return 5
	}
	return 2
}

func (m *Mob) burnsInDaylight() bool {
	return m.MobType == c.Zombie || m.MobType == c.Skeleton
}

func (m *Mob) IsPassive() bool {
	return m.MobType == c.Pig || m.MobType == c.Sheep
}

func (m *Mob) GetEntityType() c.EntityType              { return c.Mob }
func (m *Mob) GetMovementState() *c.MovementState       { return &m.MovementState }
func (m *Mob) GetPosition() (float64, float64, float64) { return m.X, m.Y, m.Z }
func (m *Mob) SetPosition(x, y, z float64)              { m.X, m.Y, m.Z = x, y, z }
func (m *Mob) GetEntityId() int32                       { return m.EntityId }
func (m *Mob) SetHP(hp int16)                           { m.HP = hp }
func (m *Mob) GetHP() int16                             { return m.HP }
func (m *Mob) GetLoggedIn() bool                        { return false }
func (m *Mob) GetDim() int32                            { return m.Dimension }
func (m *Mob) GetVelocity() (float64, float64, float64) { return m.Vx, m.Vy, m.Vz }
func (m *Mob) HasTarget() bool                          { return m.TargetId != -1 }
func (m *Mob) UnsetTarget()                             { m.TargetId = -1 }
func (m *Mob) SetTargetForced(id int32)                 { m.TargetId = id }

func (m *Mob) SetTarget(id int32) {
	if !m.HasTarget() {
		m.TargetId = id
	}
}

func (m *Mob) GetName() string {
	switch m.MobType {
	case c.Creeper:
		return "Creeper"
	case c.Skeleton:
		return "Skeleton"
	case c.Spider:
		return "Spider"
	case c.Zombie:
		return "Zombie"
	case c.Pig:
		return "Pig"
	case c.Sheep:
		return "Sheep"
	}
	return "Mob"
}

func (m *Mob) Despawn() bool {
	if m.DespawnIn < 0 {
		return false
	}
	if m.DespawnIn == 0 {
		m.DespawnIn = -1
		return true
	}
	m.DespawnIn -= 1
	return false
}

func (m *Mob) ApplyKnockback(vx, vy, vz float64) {
	m.Vx, m.Vy, m.Vz = vx, vy, vz
	m.MovementState.VelocityX, m.MovementState.VelocityY, m.MovementState.VelocityZ = vx, vy, vz
	m.MovementState.VelocityChanged = true
}

func (m *Mob) SetYawPitch(yawDeg, pitchDeg float64) {
	m.RotYaw, m.RotPitch = float32(yawDeg), float32(pitchDeg)
	m.Yaw = byte(int32(math.Floor(yawDeg*256/360)) & 0xFF)
	m.Pitch = byte(int32(math.Floor(pitchDeg*256/360)) & 0xFF)
}

// MetadataStream is the full entity metadata (flags + type specific byte), 0x7F terminated
func (m *Mob) MetadataStream() []byte {
	var flags byte
	if m.FireTicks > 0 {
		flags |= 0x01
	}
	out := []byte{0x00, flags}
	switch m.MobType {
	case c.Sheep:
		v := m.Color & 0x0F
		if m.Sheared {
			v |= 0x10
		}
		out = append(out, 0x10, v)
	case c.Creeper:
		// fuse state (1 = igniting, -1 = idle)
		out = append(out, 0x10, byte(m.creeperState), 0x11, 0)
	}
	return append(out, 0x7F)
}

func (m *Mob) TakeMetadata() []byte {
	if !m.metaDirty {
		return nil
	}
	m.metaDirty = false
	return m.MetadataStream()
}

func (m *Mob) bounds() aabb {
	w, h := m.size()
	return aabb{m.X - w/2, m.Y, m.Z - w/2, m.X + w/2, m.Y + h, m.Z + w/2}
}

func (m *Mob) Hurt(w WorldShared, dmg int16) (int16, bool) {
	if m.HP <= 0 || dmg <= 0 {
		return 0, false
	}
	fresh := true
	if m.hurtTime > mobMaxHurtTime/2 {
		if dmg <= m.lastDamage {
			return 0, false
		}
		dmg, m.lastDamage = dmg-m.lastDamage, dmg
		fresh = false
	} else {
		m.lastDamage = dmg
		m.hurtTime = mobMaxHurtTime
	}
	m.HP -= dmg
	if fresh {
		m.MovementState.IsHurt = true
	}
	if m.HP <= 0 {
		m.Die(w)
	}
	return dmg, fresh
}

func (m *Mob) Die(w WorldShared) {
	if m.DespawnIn >= 0 {
		return
	}
	if m.HP > 0 {
		m.HP = 0
	}
	m.DespawnIn = 21
	m.Vx, m.Vz = 0, 0
	m.forward, m.strafe = 0, 0

	drop := func(item int16, meta byte, count int) {
		if count > 0 {
			w.DropItemFromMinedBlock(m.X, m.Y+0.5, m.Z, item, meta, byte(count), m.Dimension, 10)
		}
	}
	switch m.MobType {
	case c.Zombie:
		drop(constants.Feather.Value, 0, rand.Intn(3))
	case c.Skeleton:
		drop(constants.Arrow.Value, 0, rand.Intn(3))
		drop(constants.Bone.Value, 0, rand.Intn(3))
	case c.Spider:
		drop(constants.String.Value, 0, rand.Intn(3))
	case c.Creeper:
		drop(constants.Gunpowder.Value, 0, rand.Intn(3))
	case c.Pig:
		pork := constants.Porkchop.Value
		if m.FireTicks > 0 {
			pork = constants.CookedPorkchop.Value
		}
		drop(pork, 0, rand.Intn(3))
	case c.Sheep:
		if !m.Sheared {
			drop(constants.Wool.Value, m.Color, 1)
		}
	}
}

// Shear drops 1-3 wool. Returns false if there was nothing to shear.
func (m *Mob) Shear(w WorldShared) bool {
	if m.MobType != c.Sheep || m.Sheared || m.HP <= 0 {
		return false
	}
	m.Sheared = true
	m.metaDirty = true
	w.DropItemFromMinedBlock(m.X, m.Y+1, m.Z, constants.Wool.Value, m.Color, byte(1+rand.Intn(3)), m.Dimension, 10)
	return true
}

func (m *Mob) Move(w WorldShared, tracker *EntityTracker) {
	if m.HP <= 0 || !w.IsLoaded(floorInt(m.X), floorInt(m.Z), m.Dimension) {
		return
	}
	m.Age++

	m.tickEnvironment(w)
	if m.HP <= 0 {
		return
	}
	if m.hurtTime > 0 {
		m.hurtTime--
	}
	if m.AttackCooldown > 0 {
		m.AttackCooldown--
	}

	if m.IsPassive() {
		m.animalAI(w)
	} else {
		m.hostileAI(w, tracker)
	}
	if m.HP <= 0 {
		return
	}

	if m.jumping {
		if m.inWater {
			m.Vy += 0.04
		} else if m.OnGround {
			m.Vy = mobJumpVel
		}
	}
	m.forward *= 0.98
	m.strafe *= 0.98

	m.tickPhysics(w)
	m.pushOthers(w)
	m.syncState()
}

func (m *Mob) syncState() {
	m.SetYawPitch(float64(m.RotYaw), float64(m.RotPitch))
	ms := &m.MovementState
	ms.X, ms.Y, ms.Z = m.X, m.Y, m.Z
	ms.Yaw, ms.Pitch = m.RotYaw, m.RotPitch
}

func (m *Mob) tickEnvironment(w WorldShared) {
	bb := m.bounds()
	burning := m.FireTicks > 0

	waterBox := aabb{bb.minX + 0.001, bb.minY + 0.4 + 0.001, bb.minZ + 0.001, bb.maxX - 0.001, bb.maxY - 0.4 - 0.001, bb.maxZ - 0.001}
	m.inWater = touches(w, m.Dimension, waterBox, func(b c.WBlock) bool { return b.IsWater() })
	if m.inWater {
		m.fallDistance = 0
		m.FireTicks = 0
		m.addWaterFlow(w, waterBox)
	}

	lavaBox := aabb{bb.minX + 0.1, bb.minY + 0.4, bb.minZ + 0.1, bb.maxX - 0.1, bb.maxY - 0.4, bb.maxZ - 0.1}
	if touches(w, m.Dimension, lavaBox, func(b c.WBlock) bool { return b.IsLava() }) {
		m.Hurt(w, 4)
		m.FireTicks = 600
	}

	if m.burnsInDaylight() && !w.IsNight() && !m.inWater && m.skyAbove(w) && rand.Float32()*30 < (1.0-0.4)*2 {
		m.FireTicks = 300
	}
	if m.FireTicks > 0 {
		if m.FireTicks%20 == 0 {
			m.Hurt(w, 1)
		}
		m.FireTicks--
	}
	if burning != (m.FireTicks > 0) {
		m.metaDirty = true
	}

	if m.headInOpaqueBlock(w) {
		m.Hurt(w, 1)
	}

	eyeY := m.Y + m.eyeHeight()
	if head := blockAt(w, floorInt(m.X), floorInt(eyeY), floorInt(m.Z), m.Dimension); head.IsWater() {
		m.air--
		if m.air <= -20 {
			m.Hurt(w, 2)
			m.air = 0
		}
	} else {
		m.air = mobMaxAir
	}
}

func (m *Mob) addWaterFlow(w WorldShared, box aabb) {
	var push vec3
	for x := floorInt(box.minX); x <= floorInt(box.maxX); x++ {
		for y := floorInt(box.minY); y <= floorInt(box.maxY); y++ {
			for z := floorInt(box.minZ); z <= floorInt(box.maxZ); z++ {
				if b := blockAt(w, x, y, z, m.Dimension); b.IsWater() {
					f := waterFlowVector(w, x, y, z, m.Dimension)
					push.x += f.x
					push.y += f.y
					push.z += f.z
				}
			}
		}
	}
	push = push.normalized()
	m.Vx += push.x * 0.014
	m.Vy += push.y * 0.014
	m.Vz += push.z * 0.014
}

func touches(w WorldShared, dim int32, bb aabb, match func(c.WBlock) bool) bool {
	for x := floorInt(bb.minX); x <= floorInt(bb.maxX); x++ {
		for y := floorInt(bb.minY); y <= floorInt(bb.maxY); y++ {
			for z := floorInt(bb.minZ); z <= floorInt(bb.maxZ); z++ {
				if match(blockAt(w, x, y, z, dim)) {
					return true
				}
			}
		}
	}
	return false
}

func (m *Mob) headInOpaqueBlock(w WorldShared) bool {
	width, _ := m.size()
	for corner := 0; corner < 8; corner++ {
		ox := (float64(corner&1) - 0.5) * width * 0.9
		oy := (float64((corner>>1)&1) - 0.5) * 0.1
		oz := (float64((corner>>2)&1) - 0.5) * width * 0.9
		if isNormalCube(blockAt(w, floorInt(m.X+ox), floorInt(m.Y+m.eyeHeight()+oy), floorInt(m.Z+oz), m.Dimension)) {
			return true
		}
	}
	return false
}

func blocksSky(b c.WBlock) bool {
	if b.TypeId == 0 || b.TypeId == byte(constants.Glass.Value) {
		return false
	}
	return b.IsLiquid() || blockTable[b.TypeId].shape != shapeNone
}

func (m *Mob) skyAbove(w WorldShared) bool {
	x, z := floorInt(m.X), floorInt(m.Z)
	for y := floorInt(m.Y + m.eyeHeight()); y < 128; y++ {
		if blocksSky(blockAt(w, x, y, z, m.Dimension)) {
			return false
		}
	}
	return true
}

// dark approximates brightness < 0.5 without lighting: night, or no sky above
func (m *Mob) dark(w WorldShared) bool {
	return w.IsNight() || !m.skyAbove(w)
}

func (m *Mob) onLadder(w WorldShared) bool {
	if m.MobType == c.Spider {
		return m.collidedH
	}
	return blockAt(w, floorInt(m.X), floorInt(m.Y), floorInt(m.Z), m.Dimension).TypeId == byte(constants.Ladder.Value)
}

func (m *Mob) applyInput(acc float32) {
	length := float32(math.Sqrt(float64(m.strafe*m.strafe + m.forward*m.forward)))
	if length < 0.01 {
		return
	}
	if length < 1 {
		length = 1
	}
	s, f := m.strafe/length, m.forward/length
	yaw := float64(m.RotYaw) * math.Pi / 180
	sin, cos := float32(math.Sin(yaw)), float32(math.Cos(yaw))
	m.Vx += float64((s*cos - f*sin) * acc)
	m.Vz += float64((f*cos + s*sin) * acc)
}

func (m *Mob) tickPhysics(w WorldShared) {
	if m.inWater {
		oldY := m.Y
		m.applyInput(0.02)
		m.move(w, m.Vx, m.Vy, m.Vz)
		m.Vx *= 0.8
		m.Vy *= 0.8
		m.Vz *= 0.8
		m.Vy -= 0.02
		if m.collidedH && m.isFree(w, m.bounds().offset(m.Vx, m.Vy+0.6-m.Y+oldY, m.Vz)) {
			m.Vy = 0.3
		}
		return
	}

	friction := float32(0.91)
	if m.OnGround {
		friction = 0.546
		below := blockAt(w, floorInt(m.X), floorInt(m.Y)-1, floorInt(m.Z), m.Dimension)
		if below.TypeId != 0 {
			friction = slipperiness(below) * 0.91
		}
	}
	accel := float32(0.16277136) / (friction * friction * friction)
	if m.OnGround {
		m.applyInput(0.1 * accel)
	} else {
		m.applyInput(0.02)
	}

	if m.onLadder(w) {
		const maxV = 0.15
		m.Vx = math.Max(-maxV, math.Min(maxV, m.Vx))
		m.Vz = math.Max(-maxV, math.Min(maxV, m.Vz))
		m.Vy = math.Max(-maxV, m.Vy)
		m.fallDistance = 0
	}

	m.move(w, m.Vx, m.Vy, m.Vz)

	if m.collidedH && m.onLadder(w) {
		m.Vy = 0.2
	}

	m.Vx *= float64(friction)
	m.Vy *= 0.98
	m.Vz *= float64(friction)
	m.Vy -= mobGravity
}

func (m *Mob) isFree(w WorldShared, bb aabb) bool {
	if len(collectCollisionBoxes(w, m.Dimension, bb)) > 0 {
		return false
	}
	return !touches(w, m.Dimension, bb, func(b c.WBlock) bool { return b.IsLiquid() })
}

func sweep(solids []aabb, bb aabb, dx, dy, dz float64) (aabb, float64, float64, float64) {
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
	return bb.offset(0, 0, dz), dx, dy, dz
}

func (m *Mob) move(w WorldShared, dx, dy, dz float64) {
	if m.inWeb {
		m.inWeb = false
		dx, dy, dz = dx*0.25, dy*0.05, dz*0.25
		m.Vx, m.Vy, m.Vz = 0, 0, 0
	}

	ox, oy, oz := dx, dy, dz
	start := m.bounds()
	solids := collectCollisionBoxes(w, m.Dimension, start.union(start.offset(dx, dy, dz)))
	bb, dx, dy, dz := sweep(solids, start, dx, dy, dz)

	canStep := m.OnGround || (oy != dy && oy < 0)
	if canStep && (ox != dx || oz != dz) {
		stepArea := start.union(start.offset(ox, mobStepHeight, oz))
		stepSolids := collectCollisionBoxes(w, m.Dimension, stepArea)
		sbb, sx, sy, sz := sweep(stepSolids, start, ox, mobStepHeight, oz)
		down := -float64(mobStepHeight)
		for _, s := range stepSolids {
			down = clipY(sbb, s, down)
		}
		sbb = sbb.offset(0, down, 0)
		if sx*sx+sz*sz > dx*dx+dz*dz {
			bb, dx, dy, dz = sbb, sx, sy, sz
		}
	}

	m.X = (bb.minX + bb.maxX) / 2
	m.Y = bb.minY
	m.Z = (bb.minZ + bb.maxZ) / 2

	m.collidedH = ox != dx || oz != dz
	m.OnGround = oy != dy && oy < 0
	if ox != dx {
		m.Vx = 0
	}
	if oy != dy {
		m.Vy = 0
	}
	if oz != dz {
		m.Vz = 0
	}

	if m.OnGround {
		if m.fallDistance > 3 {
			m.Hurt(w, int16(math.Ceil(m.fallDistance-3)))
		}
		m.fallDistance = 0
	} else if dy < 0 {
		m.fallDistance -= dy
	}

	const inset = 0.001
	for x := floorInt(bb.minX + inset); x <= floorInt(bb.maxX-inset); x++ {
		for y := floorInt(bb.minY + inset); y <= floorInt(bb.maxY-inset); y++ {
			for z := floorInt(bb.minZ + inset); z <= floorInt(bb.maxZ-inset); z++ {
				switch blockAt(w, x, y, z, m.Dimension).TypeId {
				case byte(constants.Cobweb.Value):
					m.inWeb = true
				case byte(constants.SoulSand.Value):
					m.Vx *= 0.4
					m.Vz *= 0.4
				case byte(constants.Cactus.Value):
					m.Hurt(w, 1)
				}
			}
		}
	}
}

// pushOthers is ResolveEntityPushes: nudge overlapping mobs/players apart
func (m *Mob) pushOthers(w WorldShared) {
	bb := m.bounds()
	area := aabb{bb.minX - 0.2, bb.minY, bb.minZ - 0.2, bb.maxX + 0.2, bb.maxY, bb.maxZ + 0.2}
	for _, e := range w.SnapshotEntities() {
		if e.GetEntityId() == m.EntityId || e.GetDim() != m.Dimension {
			continue
		}
		var other aabb
		var otherMob *Mob
		switch v := e.(type) {
		case *Mob:
			if v.HP <= 0 {
				continue
			}
			other, otherMob = v.bounds(), v
		case *player.Player:
			if !v.LoggedIn || v.HP <= 0 {
				continue
			}
			other = aabb{v.X - playerWidth/2, v.Y, v.Z - playerWidth/2, v.X + playerWidth/2, v.Y + playerHeight, v.Z + playerWidth/2}
		default:
			continue
		}
		if other.maxX <= area.minX || other.minX >= area.maxX || other.maxY <= area.minY || other.minY >= area.maxY ||
			other.maxZ <= area.minZ || other.minZ >= area.maxZ {
			continue
		}
		ex, _, ez := e.GetPosition()
		dx, dz := ex-m.X, ez-m.Z
		dist := math.Max(math.Abs(dx), math.Abs(dz))
		if dist < 0.01 {
			continue
		}
		dist = math.Sqrt(dist)
		dx, dz = dx/dist, dz/dist
		f := math.Min(1/dist, 1) * 0.05
		dx, dz = dx*f, dz*f
		m.Vx -= dx
		m.Vz -= dz
		if otherMob != nil {
			otherMob.Vx += dx
			otherMob.Vz += dz
		}
	}
}

func approachAngle(cur, target, max float32) float32 {
	d := float32(wrapDegrees(float64(target - cur)))
	if d > max {
		d = max
	}
	if d < -max {
		d = -max
	}
	return cur + d
}

func (m *Mob) faceTowards(x, y, z, eye float64, maxYaw, maxPitch float32) {
	dx, dz := x-m.X, z-m.Z
	desiredYaw := float32(math.Atan2(dz, dx)*180/math.Pi) - 90
	dy := (m.Y + m.eyeHeight()) - (y + eye)
	desiredPitch := float32(math.Atan2(dy, math.Sqrt(dx*dx+dz*dz)) * 180 / math.Pi)
	m.RotYaw = approachAngle(m.RotYaw, desiredYaw, maxYaw)
	m.RotPitch = approachAngle(m.RotPitch, desiredPitch, maxPitch)
}

func validTarget(w WorldShared, id, dim int32) *player.Player {
	p, ok := w.GetPlayer(id)
	if !ok || !p.LoggedIn || p.HP <= 0 || p.Dimension != dim {
		return nil
	}
	return p
}

func (m *Mob) closestPlayer(w WorldShared, maxDist float64) (*player.Player, float64) {
	var best *player.Player
	bestSq := maxDist * maxDist
	for _, p := range w.GetPlayers() {
		if !p.LoggedIn || p.HP <= 0 || p.Dimension != m.Dimension {
			continue
		}
		dx, dy, dz := p.X-m.X, p.Y-m.Y, p.Z-m.Z
		if d := dx*dx + dy*dy + dz*dz; d <= bestSq {
			best, bestSq = p, d
		}
	}
	return best, math.Sqrt(bestSq)
}

func (m *Mob) canSee(w WorldShared, x, y, z float64) bool {
	fx, fy, fz := m.X, m.Y+m.eyeHeight(), m.Z
	dx, dy, dz := x-fx, y-fy, z-fz
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
	steps := int(dist / 0.25)
	for i := 1; i < steps; i++ {
		t := float64(i) / float64(steps)
		if isNormalCube(blockAt(w, floorInt(fx+dx*t), floorInt(fy+dy*t), floorInt(fz+dz*t), m.Dimension)) {
			return false
		}
	}
	return true
}

func (m *Mob) canSeePlayer(w WorldShared, p *player.Player) bool {
	return m.canSee(w, p.X, p.Y+playerHeight*0.85, p.Z)
}

// wanderWeight scores a candidate; lighting isn't reliable so sky exposure stands in for brightness
func (m *Mob) wanderWeight(w WorldShared, x, y, z int32) float32 {
	lit := float32(-0.5)
	if !w.IsNight() && blockAt(w, x, y+1, z, m.Dimension).TypeId == 0 {
		free := true
		for yy := y; yy < 128; yy++ {
			if blocksSky(blockAt(w, x, yy, z, m.Dimension)) {
				free = false
				break
			}
		}
		if free {
			lit = 0.5
		}
	}
	if m.IsPassive() {
		if blockAt(w, x, y-1, z, m.Dimension).TypeId == byte(constants.Grass.Value) {
			return 10
		}
		return lit
	}
	return -lit
}

func (m *Mob) standable(w WorldShared, x, y, z int32) bool {
	_, h := m.size()
	if !isNormalCube(blockAt(w, x, y-1, z, m.Dimension)) {
		return false
	}
	for yy := y; float64(yy) < float64(y)+h; yy++ {
		b := blockAt(w, x, yy, z, m.Dimension)
		if len(blockBoxes(b)) > 0 || b.IsLava() {
			return false
		}
	}
	return true
}

func (m *Mob) wander(w WorldShared) {
	if !((!m.hasGoal && rand.Intn(80) == 0) || rand.Intn(80) == 0) {
		return
	}
	ox, oy, oz := floorInt(m.X), floorInt(m.Y), floorInt(m.Z)
	found := false
	var bx, by, bz int32
	best := float32(-99999)
	for i := 0; i < 10; i++ {
		x, y, z := ox+int32(rand.Intn(13)-6), oy+int32(rand.Intn(7)-3), oz+int32(rand.Intn(13)-6)
		for y > oy-4 && y > 1 && !isNormalCube(blockAt(w, x, y-1, z, m.Dimension)) {
			y--
		}
		if !m.standable(w, x, y, z) {
			continue
		}
		if wt := m.wanderWeight(w, x, y, z); wt > best {
			best, bx, by, bz, found = wt, x, y, z, true
		}
	}
	if found {
		m.setGoal(bx, by, bz)
	}
}

func (m *Mob) setGoal(x, y, z int32) {
	m.hasGoal = true
	m.goalX, m.goalY, m.goalZ = x, y, z
	m.goalTicks = 0
}

// followGoal is FollowPath with a straight line instead of a computed path
func (m *Mob) followGoal(w WorldShared) bool {
	m.jumping = false
	if m.inWater && rand.Float32() < 0.8 {
		m.jumping = true
	}
	if !m.hasGoal {
		return false
	}
	m.goalTicks++
	if rand.Intn(100) == 0 || m.goalTicks > 200 {
		m.hasGoal = false
		return false
	}

	width, _ := m.size()
	half := float64(int(width+1)) * 0.5
	dx := float64(m.goalX) + half - m.X
	dz := float64(m.goalZ) + half - m.Z
	if th := width * 2; dx*dx+dz*dz <= th*th {
		m.hasGoal = false
		if m.collidedH {
			m.jumping = true
		}
		return true
	}

	targetYaw := float32(math.Atan2(dz, dx)*180/math.Pi) - 90
	m.RotYaw = approachAngle(m.RotYaw, targetYaw, 30)
	m.forward = m.speed()

	if float64(m.goalY)-math.Floor(m.Y+0.5) > 0 || (m.collidedH && m.OnGround) {
		m.jumping = true
	}
	return true
}

func (m *Mob) idle(w WorldShared) {
	m.forward, m.strafe = 0, 0
	if rand.Float32() < 0.02 {
		if p, _ := m.closestPlayer(w, mobLookRange); p != nil {
			m.lookTarget = p.GetEntityId()
			m.lookTicks = int32(10 + rand.Intn(20))
		} else {
			m.randomYawVel = (rand.Float32() - 0.5) * 20
		}
	}
	if m.lookTarget != -1 {
		p := validTarget(w, m.lookTarget, m.Dimension)
		if p != nil {
			m.faceTowards(p.X, p.Y, p.Z, 1.62, 10, 40)
			m.lookTicks--
		}
		dx, dy, dz := 0.0, 0.0, 0.0
		if p != nil {
			dx, dy, dz = p.X-m.X, p.Y-m.Y, p.Z-m.Z
		}
		if p == nil || m.lookTicks <= 0 || dx*dx+dy*dy+dz*dz > mobLookRange*mobLookRange {
			m.lookTarget = -1
		}
		return
	}
	if rand.Float32() < 0.05 {
		m.randomYawVel = (rand.Float32() - 0.5) * 20
	}
	m.RotYaw += m.randomYawVel
	m.RotPitch = 0
}

func (m *Mob) animalAI(w WorldShared) {
	m.wander(w)
	if !m.followGoal(w) {
		m.idle(w)
	}
	m.randomYawVel *= 0.9
}

func (m *Mob) findPlayerToAttack(w WorldShared) *player.Player {
	if m.MobType == c.Spider && !m.dark(w) {
		return nil
	}
	p, _ := m.closestPlayer(w, mobAggroRange)
	if p == nil || !m.canSeePlayer(w, p) {
		return nil
	}
	return p
}

func (m *Mob) hostileAI(w WorldShared, tracker *EntityTracker) {
	m.hasAttacked = false

	target := validTarget(w, m.TargetId, m.Dimension)
	if target == nil {
		m.TargetId = -1
		if p := m.findPlayerToAttack(w); p != nil {
			m.TargetId = p.GetEntityId()
			m.setGoal(floorInt(p.X), floorInt(p.Y), floorInt(p.Z))
		}
	} else {
		dx, dy, dz := target.X-m.X, target.Y-m.Y, target.Z-m.Z
		dist := float32(math.Sqrt(dx*dx + dy*dy + dz*dz))
		if m.canSeePlayer(w, target) {
			m.tryAttack(w, target, dist, tracker)
		} else {
			m.onTargetLostSight()
		}
		if m.HP <= 0 {
			return
		}
		target = validTarget(w, m.TargetId, m.Dimension)
	}

	if m.MobType == c.Creeper && target == nil && m.fuse > 0 {
		m.defuse()
	}

	if m.hasAttacked || target == nil || (m.hasGoal && rand.Intn(20) != 0) {
		if !m.hasAttacked {
			m.wander(w)
		}
	} else {
		m.setGoal(floorInt(target.X), floorInt(target.Y), floorInt(target.Z))
	}

	m.RotPitch = 0
	if m.followGoal(w) {
		if m.hasAttacked && target != nil {
			prev := m.RotYaw
			m.RotYaw = float32(math.Atan2(target.Z-m.Z, target.X-m.X)*180/math.Pi) - 90
			diff := float64(prev-m.RotYaw+90) * math.Pi / 180
			fwd := m.forward
			m.strafe = -float32(math.Sin(diff)) * fwd
			m.forward = float32(math.Cos(diff)) * fwd
		}
		if target != nil {
			m.faceTowards(target.X, target.Y, target.Z, 1.62, 30, 30)
		}
	} else {
		m.idle(w)
	}
	m.randomYawVel *= 0.9
}

func (m *Mob) tryAttack(w WorldShared, p *player.Player, dist float32, tracker *EntityTracker) {
	switch m.MobType {
	case c.Creeper:
		primed := m.creeperState > 0
		if (!primed && dist < 3) || (primed && dist < 7) {
			if m.creeperState != 1 {
				m.creeperState = 1
				m.metaDirty = true
			}
			m.fuse++
			if m.fuse >= 30 {
				m.explode(w, tracker, 3)
				return
			}
			m.hasAttacked = true
		} else {
			m.defuse()
		}
		return
	case c.Spider:
		if !m.dark(w) && rand.Intn(100) == 0 {
			m.TargetId = -1
			return
		}
		if dist > 2 && dist < 6 && rand.Intn(10) == 0 && m.OnGround {
			dx, dz := p.X-m.X, p.Z-m.Z
			d := math.Sqrt(dx*dx + dz*dz)
			m.Vx = dx/d*0.5*0.8 + m.Vx*0.2
			m.Vz = dz/d*0.5*0.8 + m.Vz*0.2
			m.Vy = 0.4
			return
		}
	}

	_, h := m.size()
	if m.AttackCooldown <= 0 && dist < 2 && p.Y+playerHeight > m.Y && p.Y < m.Y+h {
		m.AttackCooldown = 20
		if w.HurtPlayer(p, m, m.attackStrength()) <= 0 {
			m.TargetId = -1
			tracker.ResetViewer(w, p.GetEntityId())
		}
	}
}

func (m *Mob) onTargetLostSight() {
	if m.MobType == c.Creeper && m.fuse > 0 {
		m.defuse()
	}
}

func (m *Mob) defuse() {
	if m.creeperState != -1 {
		m.creeperState = -1
		m.metaDirty = true
	}
	if m.fuse > 0 {
		m.fuse--
	}
}

func (m *Mob) explode(w WorldShared, tracker *EntityTracker, size float64) {
	cx, cy, cz := m.X, m.Y, m.Z

	// The client only spawns explosion particles for blocks listed in the packet
	destroyed := explosionBlocks(w, m.Dimension, cx, cy, cz, float32(size))
	tracker.SendToViewers(w, m.EntityId, explosionPacket(cx, cy, cz, float32(size), destroyed))
	air := c.NewAirBlock()
	oldTypes := make([]byte, len(destroyed))
	for i, pos := range destroyed {
		oldTypes[i] = w.GetBlock(pos[0], byte(pos[1]), pos[2], m.Dimension).TypeId
		w.SetBlockInQueue(pos[0], pos[1], pos[2], air, m.Dimension)
	}
	// Notify only once every block is gone so neighbours see the final crater
	for i, pos := range destroyed {
		w.NotifyBlockRemoved(pos[0], pos[1], pos[2], oldTypes[i], m.Dimension)
	}

	m.HP = 0
	tracker.SendToViewers(w, m.EntityId, w.DespawnEntity(m.EntityId))
	w.RemoveEntity(m.EntityId)
	tracker.ResetEntity(m.EntityId)

	r := size * 2
	impactOf := func(x, y, z, midY float64) (float64, float64, float64, float64) {
		dx, dy, dz := x-cx, y-cy, z-cz
		d := math.Sqrt(dx*dx + dy*dy + dz*dz)
		frac := d / r
		if frac > 1 || d == 0 {
			return 0, 0, 0, 0
		}
		density := 0.0
		if m.canSee(w, x, midY, z) || d < 1 {
			density = 1
		}
		return (1 - frac) * density, dx / d, dy / d, dz / d
	}
	damage := func(impact float64) int16 {
		return int16((impact*impact+impact)/2*8*r + 1)
	}

	for _, e := range w.SnapshotEntities() {
		if e.GetDim() != m.Dimension || e.GetEntityId() == m.EntityId {
			continue
		}
		switch v := e.(type) {
		case *player.Player:
			if !v.LoggedIn || v.HP <= 0 {
				continue
			}
			impact, nx, ny, nz := impactOf(v.X, v.Y, v.Z, v.Y+playerHeight/2)
			if impact <= 0 {
				continue
			}
			if w.HurtPlayer(v, nil, damage(impact)) <= 0 {
				tracker.ResetViewer(w, v.GetEntityId())
				continue
			}
			ms := c.MovementState{VelocityX: nx * impact, VelocityY: ny * impact, VelocityZ: nz * impact}
			v.Connection.Write(w.NewEntityVelocityPacket(v.GetEntityId(), ms))
		case *Mob:
			_, h := v.size()
			impact, nx, ny, nz := impactOf(v.X, v.Y, v.Z, v.Y+h/2)
			if impact <= 0 {
				continue
			}
			v.Hurt(w, damage(impact))
			v.ApplyKnockback(v.Vx+nx*impact, v.Vy+ny*impact, v.Vz+nz*impact)
		}
	}
}

// blastResistance holds vanilla Beta 1.7.3 resistance for blocks that resist better than instant-break
var blastResistance = map[byte]float32{
	byte(c.Stone.Value):             30,
	byte(c.Grass.Value):             3,
	byte(c.Dirt.Value):              2.5,
	byte(c.Cobblestone.Value):       30,
	byte(c.Planks.Value):            15,
	byte(c.Bedrock.Value):           6000000,
	byte(c.WaterFlowing.Value):      500,
	byte(c.WaterStill.Value):        500,
	byte(c.LavaFlowing.Value):       500,
	byte(c.LavaStill.Value):         500,
	byte(c.Sand.Value):              2.5,
	byte(c.Gravel.Value):            3,
	byte(c.GoldOre.Value):           15,
	byte(c.IronOre.Value):           15,
	byte(c.CoalOre.Value):           15,
	byte(c.Log.Value):               10,
	byte(c.Leaves.Value):            1,
	byte(c.Sponge.Value):            3,
	byte(c.Glass.Value):             1.5,
	byte(c.LapisLazuliOre.Value):    15,
	byte(c.LapisLazuliBlock.Value):  30,
	byte(c.Dispenser.Value):         15,
	byte(c.Sandstone.Value):         4,
	byte(c.Noteblock.Value):         4,
	byte(c.Wool.Value):              4,
	byte(c.GoldBlock.Value):         30,
	byte(c.IronBlock.Value):         30,
	byte(c.DoubleStoneSlab.Value):   30,
	byte(c.StoneSlab.Value):         30,
	byte(c.Bricks.Value):            30,
	byte(c.Bookshelf.Value):         7.5,
	byte(c.MossyCobblestone.Value):  30,
	byte(c.Obsidian.Value):          6000,
	byte(c.MonsterSpawner.Value):    25,
	byte(c.WoodenStairs.Value):      15,
	byte(c.Chest.Value):             12.5,
	byte(c.DiamondOre.Value):        15,
	byte(c.DiamondBlock.Value):      30,
	byte(c.CraftingTable.Value):     12.5,
	byte(c.Furnace.Value):           17.5,
	byte(c.FurnaceLit.Value):        17.5,
	byte(c.WoodenDoor.Value):        15,
	byte(c.Ladder.Value):            2,
	byte(c.CobblestoneStairs.Value): 30,
	byte(c.IronDoor.Value):          25,
	byte(c.RedstoneOreOff.Value):    15,
	byte(c.RedstoneOreOn.Value):     15,
	byte(c.SnowLayer.Value):         0.5,
	byte(c.Ice.Value):               2.5,
	byte(c.SnowBlock.Value):         1,
	byte(c.Clay.Value):              3,
	byte(c.Jukebox.Value):           30,
	byte(c.Netherrack.Value):        2,
	byte(c.SoulSand.Value):          2.5,
	byte(c.Glowstone.Value):         1.5,
}

// explosionBlocks ray-casts the outer shell of the blast like vanilla and returns the blocks it destroys
func explosionBlocks(w WorldShared, dim int32, cx, cy, cz float64, size float32) [][3]int32 {
	const grid = 16
	seen := make(map[[3]int32]bool)
	var out [][3]int32
	for gx := 0; gx < grid; gx++ {
		for gy := 0; gy < grid; gy++ {
			for gz := 0; gz < grid; gz++ {
				if gx != 0 && gx != grid-1 && gy != 0 && gy != grid-1 && gz != 0 && gz != grid-1 {
					continue
				}
				dx := float64(gx)/(grid-1)*2 - 1
				dy := float64(gy)/(grid-1)*2 - 1
				dz := float64(gz)/(grid-1)*2 - 1
				l := math.Sqrt(dx*dx + dy*dy + dz*dz)
				dx, dy, dz = dx/l, dy/l, dz/l
				power := size * (0.7 + rand.Float32()*0.6)
				x, y, z := cx, cy, cz
				for ; power > 0; power -= 0.3 * 0.75 {
					bx, by, bz := floorInt(x), floorInt(y), floorInt(z)
					if by >= 0 && by < 128 && w.IsLoaded(bx, bz, dim) {
						if b := w.GetBlock(bx, byte(by), bz, dim); b.TypeId != 0 {
							power -= (blastResistance[b.TypeId]*3/5 + 0.3) * 0.3
						}
						if power > 0 {
							key := [3]int32{bx, by, bz}
							if !seen[key] {
								seen[key] = true
								if w.GetBlock(bx, byte(by), bz, dim).TypeId != 0 {
									out = append(out, key)
								}
							}
						}
					}
					x += dx * 0.3
					y += dy * 0.3
					z += dz * 0.3
				}
			}
		}
	}
	return out
}

func explosionPacket(x, y, z float64, radius float32, blocks [][3]int32) []byte {
	var b bytes.Buffer
	b.WriteByte(0x3C)
	binary.Write(&b, binary.BigEndian, x)
	binary.Write(&b, binary.BigEndian, y)
	binary.Write(&b, binary.BigEndian, z)
	binary.Write(&b, binary.BigEndian, radius)
	binary.Write(&b, binary.BigEndian, int32(len(blocks)))
	ox, oy, oz := floorInt(x), floorInt(y), floorInt(z)
	for _, p := range blocks {
		b.WriteByte(byte(int8(p[0] - ox)))
		b.WriteByte(byte(int8(p[1] - oy)))
		b.WriteByte(byte(int8(p[2] - oz)))
	}
	return b.Bytes()
}
