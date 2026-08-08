package level

import (
	"log"
	"math"
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

	OnGround bool
}

func (m *Mob) GetName() string {
	return "Mob"
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
		HP:        20,
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

func (m *Mob) Move(w *World) {
	if !m.HasTarget() {
		return
	}

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
		w.BroadcastMobPositionAndRotation(m, mx, my, mz, yaw, pitch)
		w.BroadcastEntityVelocity(m.EntityId, 0, 0, 0)
		m.SetYawPitch(yaw, pitch)
		m.Vx, m.Vy, m.Vz = 0, 0, 0
		return
	}

	horizDist := math.Sqrt(dx*dx + dz*dz)
	speed := m.Speed()

	var vx, vz float64
	if horizDist > 0.0001 {
		vx = (dx / horizDist) * speed
		vz = (dz / horizDist) * speed
	}

	blockedX, blockedZ := m.checkObstacles(w, mx, my, mz, vx, vz)
	if blockedX {
		vx = 0
	}
	if blockedZ {
		vz = 0
	}

	vy := m.resolveVerticalVelocity(w, mx, my, mz, vx, vz, blockedX || blockedZ)

	// log.Printf("Spider debug dx=%.3f dy=%.3f dz=%.3f dist=%.3f horizDist=%.3f blockedX=%v blockedZ=%v onGround=%v (id=%d)",
	// 	dx, dy, dz, dist, horizDist, blockedX, blockedZ, m.OnGround, m.EntityId)

	newX := mx + vx
	newY := my + vy
	newZ := mz + vz

	belowBlock := w.GetBlock(int32(math.Floor(newX)), byte(math.Floor(newY-0.01)), int32(math.Floor(newZ)), m.Dimension)
	onGround := belowBlock.IsSolid() && vy <= 0
	if onGround {
		newY = math.Floor(newY)
		vy = 0
	}

	w.BroadcastMobPositionAndRotation(m, newX, newY, newZ, yaw, pitch)
	w.BroadcastEntityVelocity(m.EntityId, vx, vy, vz)

	m.OnGround = onGround
	m.Vx, m.Vy, m.Vz = vx, vy, vz
	m.SetPosition(newX, newY, newZ)
	m.SetYawPitch(yaw, pitch)
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

	if m.OnGround {
		if blockedAhead {
			bx := int32(math.Floor(x + vx))
			by := byte(math.Floor(y))
			bz := int32(math.Floor(z + vz))
			b2 := w.GetBlock(bx, by+1, bz, m.Dimension)

			if !b2.IsSolid() {
				return jumpVelocity 
			}
			if m.MobType == 52 { 
				return climbSpeed
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
		return 2.0
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

func (w *World) SpawnSpider(x, y, z, dim int32, target int32) {
	log.Printf("Spawned Spider at x=%d, y=%d, z=%d", x, y, z)
	s := NewSpider(w, float64(x), float64(y), float64(z), dim)
	s.SetTarget(target)
	w.Entities[s.EntityId] = s
	w.BroadcastMobSpawn(s.MobType, s.Metadata, x, y, z, s.Yaw, s.Pitch, s.Dimension, s.EntityId)
}
