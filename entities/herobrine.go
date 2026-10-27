package entities

import (
	"math"

	c "github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/player"
)

const (
	HerobrineName = "Herobrine"
	// Beta clients fetch skins by username, so he is spawned under the account whose skin he wears
	HerobrineSkin = "null"

	herobrineLifetime  = 20 * 90
	herobrineNearRange = 12.0
	herobrineGazeTicks = 6
	herobrineGazeCos   = 0.97
	playerEyeHeight    = 1.62

	lurkLifetime   = 20 * 25
	lurkNearRange  = 1.5
	lurkGazeTicks  = 10
	stalkLifetime  = 20 * 60
	stalkNoticeCos = 0.8
	stalkReach     = 1.8
	stalkDamage    = 6
	scareLifetime  = 20 * 3

	possessedLifetime = 20 * 60
	possessedKeepAway = 3.0
)

type HerobrineMode byte

const (
	// HerobrineWatch stands far away and vanishes when approached or stared at
	HerobrineWatch HerobrineMode = iota
	// HerobrineLurk shows up close by and vanishes once he has been seen
	HerobrineLurk
	// HerobrineStalk creeps up while unseen and strikes if not noticed in time
	HerobrineStalk
	// HerobrineScare stands right in front of the player for a moment
	HerobrineScare
)

func NewHerobrine(id, target int32, x, y, z float64, dim int32, mode HerobrineMode) *Mob {
	m := NewMob(id, c.Herobrine, x, y, z, dim)
	m.TargetId = target
	m.HerobrineMode = mode
	return m
}

func (m *Mob) IsHerobrine() bool { return m.MobType == c.Herobrine }

// VisibleTo limits Herobrine to the one player he haunts
func (m *Mob) VisibleTo(viewerId int32) bool {
	return !m.IsHerobrine() || viewerId == m.TargetId
}

// InSightOf reports whether p is looking at the mob (within the cone given by minCos) with nothing in between
func (m *Mob) InSightOf(w WorldShared, p *player.Player, minCos float64) bool {
	yaw, pitch := float64(p.Yaw)*math.Pi/180, float64(p.Pitch)*math.Pi/180
	lx, ly, lz := -math.Sin(yaw)*math.Cos(pitch), -math.Sin(pitch), math.Cos(yaw)*math.Cos(pitch)
	ex, ey, ez := p.X, p.Y+playerEyeHeight, p.Z
	tx, ty, tz := m.X-ex, m.Y+m.eyeHeight()-ey, m.Z-ez
	d := math.Sqrt(tx*tx + ty*ty + tz*tz)
	if d == 0 {
		return true
	}
	return (lx*tx+ly*ty+lz*tz)/d > minCos && m.canSee(w, ex, ey, ez)
}

func (m *Mob) FaceTowardsPlayer(p *player.Player) {
	m.faceTowards(p.X, p.Y, p.Z, playerEyeHeight, 360, 90)
	m.syncState()
}

func (m *Mob) herobrineAI(w WorldShared, tracker *EntityTracker) {
	m.forward, m.strafe, m.jumping = 0, 0, false
	p := validTarget(w, m.TargetId, m.Dimension)
	if p == nil {
		m.Vanished = true
		return
	}
	dx, dy, dz := p.X-m.X, p.Y-m.Y, p.Z-m.Z
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)

	switch m.HerobrineMode {
	case HerobrineWatch:
		m.faceTowards(p.X, p.Y, p.Z, playerEyeHeight, 4, 4)
		m.vanishWhen(w, p, herobrineLifetime, dist < herobrineNearRange, herobrineGazeCos, herobrineGazeTicks)
	case HerobrineLurk:
		m.faceTowards(p.X, p.Y, p.Z, playerEyeHeight, 10, 10)
		m.vanishWhen(w, p, lurkLifetime, dist < lurkNearRange, 0.9, lurkGazeTicks)
	case HerobrineStalk:
		m.stalk(w, tracker, p, dist)
	case HerobrineScare:
		m.faceTowards(p.X, p.Y, p.Z, playerEyeHeight, 360, 90)
		if m.Age > scareLifetime {
			m.Vanished = true
		}
	}
}

func (m *Mob) vanishWhen(w WorldShared, p *player.Player, lifetime int32, tooClose bool, gazeCos float64, gazeTicks int32) {
	if m.Age > lifetime || tooClose {
		m.Vanished = true
		return
	}
	if m.InSightOf(w, p, gazeCos) {
		m.gaze++
	} else {
		m.gaze = 0
	}
	if m.gaze >= gazeTicks {
		m.Vanished = true
	}
}

func (m *Mob) stalk(w WorldShared, tracker *EntityTracker, p *player.Player, dist float64) {
	if m.Age > stalkLifetime || p.IsCreative {
		m.Vanished = true
		return
	}
	m.faceTowards(p.X, p.Y, p.Z, playerEyeHeight, 360, 90)
	if m.InSightOf(w, p, stalkNoticeCos) {
		m.gaze++
		if m.gaze >= 3 {
			m.Vanished = true
		}
		return
	}
	m.gaze = 0
	if dist < stalkReach {
		if w.HurtPlayer(p, m, stalkDamage) <= 0 {
			tracker.ResetViewer(w, p.GetEntityId())
		}
		m.Vanished = true
		return
	}
	m.forward = 1.0
	if m.collidedH && m.OnGround {
		m.jumping = true
	}
}

func (m *Mob) IsPossessed() bool { return m.possessedBy != -1 && m.possessedTicks > 0 }

// Possess makes a passive mob stare at and silently follow p for a while
func (m *Mob) Possess(p *player.Player) {
	m.possessedBy = p.GetEntityId()
	m.possessedTicks = possessedLifetime
	m.hasGoal = false
}

func (m *Mob) possessedAI(w WorldShared) {
	m.forward, m.strafe, m.jumping = 0, 0, false
	m.possessedTicks--
	p := validTarget(w, m.possessedBy, m.Dimension)
	if p == nil || m.possessedTicks <= 0 {
		m.possessedBy, m.possessedTicks = -1, 0
		return
	}
	m.faceTowards(p.X, p.Y, p.Z, playerEyeHeight, 360, 90)
	dx, dz := p.X-m.X, p.Z-m.Z
	if dx*dx+dz*dz < possessedKeepAway*possessedKeepAway || m.InSightOf(w, p, 0.6) {
		return
	}
	m.forward = m.speed()
	if (m.collidedH && m.OnGround) || m.inWater {
		m.jumping = true
	}
}
