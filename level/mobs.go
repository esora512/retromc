package level

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/leNicDev/retromc/player"
)

const groundFeetOffset = 0.75

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

	WanderDirX, WanderDirZ float64
	WanderTicksLeft        int32
	KnockbackTicks         int32
}

func (m *Mob) ApplyKnockback(vx, vy, vz float64) {
	m.Vx, m.Vy, m.Vz = vx, vy, vz
	m.KnockbackTicks = 6
}

func (m *Mob) GetName() string {
	return fmt.Sprintf("Entity %d", m.EntityId)
}

func (m *Mob) GetPosition() (float64, float64, float64) {
	return m.X, m.Y, m.Z
}

func (m *Mob) SetPosition(x, y, z float64) {
	m.X, m.Y, m.Z = x, y, z
}

func (m *Mob) IsRideable() bool {
	return false
}

func (m *Mob) GetEntityId() int32 {
	return m.EntityId
}

func (m *Mob) IsPlayer() bool {
	return false
}

func (m *Mob) SetHP(hp int16) {
	m.HP = hp
}

func (m *Mob) GetLoggedIn() bool {
	return false
}

func (m *Mob) GetDim() int32 {
	return m.Dimension
}

func (m *Mob) GetHP() int16 {
	return m.HP
}

func NewSpider(w *World, x, y, z float64, dim int32) *Mob {
	m := Mob{EntityId: w.NextEntityId(),
		X: x, Y: y, Z: z,
		Yaw: 0, Pitch: 0,
		Vx: 0, Vy: 0, Vz: 0,
		Dimension: dim,
		MobType:   52,
		Metadata:  0,
		TargetId:  -1,
		HP:        10,
		OnGround:  true,
	}
	return &m
}

func (m *Mob) HasTarget() bool {
	return m.TargetId != -1
}

func (m *Mob) SetTarget(id int32) {
	if !m.HasTarget() {
		m.TargetId = id
	}
}

func (m *Mob) UnsetTarget() {
	m.TargetId = -1
}

func (m *Mob) wander(w *World) {
	if m.WanderTicksLeft <= 0 {
		m.pickNewWanderDirection()
	}
	m.WanderTicksLeft--

	mx, my, mz := m.GetPosition()
	speed := m.Speed() * 0.4

	vx := m.WanderDirX * speed
	vz := m.WanderDirZ * speed

	avoidX, avoidZ := m.adjustForOthers(w)
	vx += avoidX * speed
	vz += avoidZ * speed
	if mag := math.Sqrt(vx*vx + vz*vz); mag > speed {
		vx = (vx / mag) * speed
		vz = (vz / mag) * speed
	}

	blockedX, blockedZ := m.checkObstacles(w, mx, my, mz, vx, vz)
	if blockedX {
		vx = 0
	}
	if blockedZ {
		vz = 0
	}

	if blockedX || blockedZ {
		m.WanderTicksLeft = 0
	}

	vy := m.resolveVerticalVelocity(w, mx, my, mz, vx, vz, blockedX || blockedZ)

	newX := mx + vx
	newY := my + vy
	newZ := mz + vz

	belowBlock := w.GetBlock(int32(math.Floor(newX)), byte(math.Floor(newY-0.01)), int32(math.Floor(newZ)), m.Dimension)
	onGround := belowBlock.IsSolid() && vy <= 0
	newY, vy, onGround = m.resolveGroundCollision(w, newX, newY, newZ, vy)

	yaw, pitch := computeYawPitch(vx, 0, vz)
	if vx == 0 && vz == 0 {
		yaw, pitch = float64(m.Yaw)*360/256, float64(m.Pitch)*360/256
	}

	w.MulticastMobPositionAndRotation(m, newX, newY, newZ, yaw, pitch)
	//w.MulticastEntityVelocity(m.EntityId, vx, vy, vz)

	m.OnGround = onGround
	m.Vx, m.Vy, m.Vz = vx, vy, vz
	m.SetPosition(newX, newY, newZ)
	m.SetYawPitch(yaw, pitch)
}

func (m *Mob) pickNewWanderDirection() {
	angle := rand.Float64() * 2 * math.Pi
	m.WanderDirX = math.Cos(angle)
	m.WanderDirZ = math.Sin(angle)
	m.WanderTicksLeft = int32(40 + rand.Intn(80)) // 2-6 seconds at 20 ticks/sec
}

func (m *Mob) Move(w *World) {
	if m.KnockbackTicks > 0 {
		m.tickKnockback(w)
		return
	}

	night := w.IsNight()

	if night && !m.HasTarget() {
		if pid, found := m.findNearbyPlayer(w); found {
			m.SetTarget(pid)
		}
	}

	if m.HasTarget() {
		m.moveTowardTarget(w)
		return
	}

	m.wander(w)
}

func (m *Mob) SetTargetForced(id int32) {
	m.TargetId = id
}

func (m *Mob) tickKnockback(w *World) {
	const drag = 0.91
	const gravity = 0.08
	const terminalVelocity = -3.92

	mx, my, mz := m.GetPosition()

	vx, vy, vz := m.Vx, m.Vy, m.Vz

	if !m.OnGround {
		vy -= gravity
		if vy < terminalVelocity {
			vy = terminalVelocity
		}
	}

	newX := mx + vx
	newY := my + vy
	newZ := mz + vz

	belowBlock := w.GetBlock(int32(math.Floor(newX)), byte(math.Floor(newY-0.01)), int32(math.Floor(newZ)), m.Dimension)
	onGround := belowBlock.IsSolid() && vy <= 0
	newY, vy, onGround = m.resolveGroundCollision(w, newX, newY, newZ, vy)

	vx *= drag
	vz *= drag

	yaw, pitch := float64(m.Yaw)*360/256, float64(m.Pitch)*360/256

	w.MulticastMobPositionAndRotation(m, newX, newY, newZ, yaw, pitch)
	//w.MulticastEntityVelocity(m.EntityId, vx, vy, vz)

	m.OnGround = onGround
	m.Vx, m.Vy, m.Vz = vx, vy, vz
	m.SetPosition(newX, newY, newZ)
	m.KnockbackTicks--
}

func (m *Mob) findNearbyPlayer(w *World) (int32, bool) {
	const detectionRadius = 16.0

	mx, my, mz := m.GetPosition()

	var closestId int32 = -1
	closestDist := math.MaxFloat64

	for _, e := range w.Players {
		if !e.GetLoggedIn() {
			continue
		}
		if e.GetDim() != m.Dimension {
			continue
		}
		px, py, pz := e.GetPosition()
		dx := px - mx
		dy := py - my
		dz := pz - mz
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
		if dist <= detectionRadius && dist < closestDist {
			closestDist = dist
			closestId = e.GetEntityId()
		}
	}

	if closestId == -1 {
		return 0, false
	}
	return closestId, true
}


func (m *Mob) resolveGroundCollision(w *World, newX, newY, newZ, vy float64) (float64, float64, bool) {
	if vy > 0 {
		return newY, vy, false
	}

	bx := int32(math.Floor(newX))
	bz := int32(math.Floor(newZ))

	feetBlockY := int32(math.Floor(newY))
	b := w.GetBlock(bx, byte(feetBlockY), bz, m.Dimension)
	if b.IsSolid() {
		return float64(feetBlockY) + groundFeetOffset, 0, true
	}

	supportBlockY := int32(math.Floor(newY - 0.01))
	b = w.GetBlock(bx, byte(supportBlockY), bz, m.Dimension)
	if b.IsSolid() {
		return float64(supportBlockY) + groundFeetOffset, 0, true
	}

	return newY, vy, false
}

func (m *Mob) moveTowardTarget(w *World) {
	t, exists := w.GetEntity(m.TargetId)
	if !exists {
		m.UnsetTarget()
		return
	}

	mx, my, mz := m.GetPosition()
	tx, ty, tz := t.GetPosition()

	dx := tx - mx
	dy := ty - my
	dz := tz - mz

	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
	yaw, pitch := computeYawPitch(dx, dy, dz)

	if dist < m.StopDistance() {
		if m.AttackCooldown > 0 {
			m.AttackCooldown--
		} else {
			m.performAttack(w, t, dx, dy, dz)
			m.AttackCooldown = m.AttackSpeed()
		}
		m.SetYawPitch(yaw, pitch)
		return
	}

	horizDist := math.Sqrt(dx*dx + dz*dz)
	speed := m.Speed()

	var vx, vz float64
	if horizDist > 0.0001 {
		vx = (dx / horizDist) * speed
		vz = (dz / horizDist) * speed
	}

	avoidX, avoidZ := m.adjustForOthers(w)
	vx += avoidX * speed
	vz += avoidZ * speed
	if mag := math.Sqrt(vx*vx + vz*vz); mag > speed {
		vx = (vx / mag) * speed
		vz = (vz / mag) * speed
	}

	blockedX, blockedZ := m.checkObstacles(w, mx, my, mz, vx, vz)
	if blockedX {
		vx = 0
	}
	if blockedZ {
		vz = 0
	}

	vy := m.resolveVerticalVelocity(w, mx, my, mz, vx, vz, blockedX || blockedZ)

	newX := mx + vx
	newY := my + vy
	newZ := mz + vz

	belowBlock := w.GetBlock(int32(math.Floor(newX)), byte(math.Floor(newY-0.01)), int32(math.Floor(newZ)), m.Dimension)
	onGround := belowBlock.IsSolid() && vy <= 0
	newY, vy, onGround = m.resolveGroundCollision(w, newX, newY, newZ, vy)

	w.MulticastMobPositionAndRotation(m, newX, newY, newZ, yaw, pitch)
	//w.MulticastEntityVelocity(m.EntityId, vx, vy, vz)

	m.OnGround = onGround
	m.Vx, m.Vy, m.Vz = vx, vy, vz
	m.SetPosition(newX, newY, newZ)
	m.SetYawPitch(yaw, pitch)
}

func (m *Mob) performAttack(w *World, t Entity, dx, dy, dz float64) {
	const lungeSpeed = 0.55
	const lungeUp = 0.50

	mx, my, mz := m.GetPosition()
	yaw, pitch := computeYawPitch(dx, dy, dz)

	horizDist := math.Sqrt(dx*dx + dz*dz)
	var vx, vz float64
	if horizDist > 0.0001 {
		vx = (dx / horizDist) * lungeSpeed
		vz = (dz / horizDist) * lungeSpeed
	}
	vy := lungeUp
	if m.OnGround {
	} else {
		vy = 0
	}

	newX := mx + vx
	newY := my + vy
	newZ := mz + vz

	w.MulticastMobPositionAndRotation(m, newX, newY, newZ, yaw, pitch)
	w.MulticastEntityVelocity(m.EntityId, vx, vy, vz)

	m.Vx, m.Vy, m.Vz = vx, vy, vz
	m.SetPosition(newX, newY, newZ)
	m.OnGround = false

	if pd, ok := t.(interface{ Damage(int16) }); ok {
		pd.Damage(m.AttackDamage())
	}

	oldHP := t.GetHP()
	newHP := oldHP - m.AttackDamage()
	t.SetHP(newHP)

	if pl, ok := t.(*player.Player); ok {
		w.SendSetHealth(pl.Connection, uint16(newHP))
	}
	w.BroadcastPain(t.GetEntityId())

	if newHP <= 0 {
		m.UnsetTarget()
	}
}

func (m *Mob) GetVelocity() (float64, float64, float64) {
	return m.Vx, m.Vy, m.Vz
}

func (m *Mob) IsMob() bool {
	return true
}

func (m *Mob) AttackSpeed() int32 {
	switch m.MobType {
	case 52:
		return 20
	default:
		return 20
	}
}

func (m *Mob) AttackDamage() int16 {
	switch m.MobType {
	case 52:
		return 2
	default:
		return 1
	}
}

func (m *Mob) checkObstacles(w *World, x, y, z float64, vx, vz float64) (blockedX, blockedZ bool) {
	const lookAhead = 0.3

	if vx != 0 {
		bx := int32(math.Floor(x + math.Copysign(lookAhead, vx)))
		by := byte(math.Floor(y))
		bz := int32(math.Floor(z))

		b1 := w.GetBlock(bx, by, bz, m.Dimension)
		b2 := w.GetBlock(bx, by+1, bz, m.Dimension)
		blockedX = b1.IsSolid() || b2.IsSolid()
	}

	if vz != 0 {
		bx := int32(math.Floor(x))
		by := byte(math.Floor(y))
		bz := int32(math.Floor(z + math.Copysign(lookAhead, vz)))

		b1 := w.GetBlock(bx, by, bz, m.Dimension)
		b2 := w.GetBlock(bx, by+1, bz, m.Dimension)
		blockedZ = b1.IsSolid() || b2.IsSolid()
	}

	return blockedX, blockedZ
}

func (m *Mob) resolveVerticalVelocity(w *World, x, y, z float64, vx, vz float64, blockedAhead bool) float64 {
	const jumpVelocity = 0.42
	const climbSpeed = 0.2
	const gravity = 0.08
	const terminalVelocity = -3.92

	if blockedAhead && m.MobType == 52 {
		return climbSpeed
	}

	if m.OnGround {
		if blockedAhead {
			bx := int32(math.Floor(x + vx))
			by := byte(math.Floor(y))
			bz := int32(math.Floor(z + vz))
			b2 := w.GetBlock(bx, by+1, bz, m.Dimension)

			if !b2.IsSolid() {
				return jumpVelocity
			}
			return 0
		}
		return 0
	}

	vy := m.Vy - gravity
	if vy < terminalVelocity {
		vy = terminalVelocity
	}
	return vy
}

func computeYawPitch(dx, dy, dz float64) (yaw, pitch float64) {
	horizDist := math.Sqrt(dx*dx + dz*dz)
	yaw = math.Atan2(-dx, dz) * (180 / math.Pi)
	pitch = math.Atan2(-dy, horizDist) * (180 / math.Pi)
	return yaw, pitch
}

func (m *Mob) SetYawPitch(yawDeg, pitchDeg float64) {
	m.Yaw = byte(int32(yawDeg*256/360) & 0xFF)
	m.Pitch = byte(int32(pitchDeg*256/360) & 0xFF)
}

func (m *Mob) StopDistance() float64 {
	switch m.MobType {
	case 52:
		return 1.25
	default:
		return 1.5
	}
}

func (m *Mob) Speed() float64 {
	switch m.MobType {
	case 52:
		return 0.3
	default:
		return 0.2
	}
}

func (w *World) SpawnSpider(x, y, z, dim int32, target int32) int32 {
	s := NewSpider(w, float64(x), float64(y), float64(z), dim)
	s.SetTarget(target)
	w.Entities[s.EntityId] = s
	w.BroadcastMobSpawn(s.MobType, s.Metadata, x, y, z, s.Yaw, s.Pitch, s.Dimension, s.EntityId)
	return s.EntityId
}

func (m *Mob) adjustForOthers(w *World) (avoidX, avoidZ float64) {
	const minSeparation = 1.5
	const maxCrowd = 16

	mx, my, mz := m.GetPosition()

	nearby := 0
	for _, e := range w.Entities {
		if e.GetEntityId() == m.EntityId || e.IsPlayer() {
			continue
		}
		if e.GetDim() != m.Dimension {
			continue
		}

		ex, ey, ez := e.GetPosition()

		if math.Abs(ey-my) > 1.5 {
			continue
		}

		dx := mx - ex
		dz := mz - ez
		dist := math.Sqrt(dx*dx + dz*dz)

		if dist >= minSeparation {
			continue
		}

		nearby++

		if dist < 0.0001 {
			angle := float64(m.EntityId%360) * (math.Pi / 180)
			dx = math.Cos(angle)
			dz = math.Sin(angle)
			dist = 1
		}

		strength := (minSeparation - dist) / minSeparation
		avoidX += (dx / dist) * strength
		avoidZ += (dz / dist) * strength
	}

	if nearby >= maxCrowd {
		return 0, 0
	}

	return avoidX, avoidZ
}
