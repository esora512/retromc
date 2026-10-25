package entities

import (
	"math"

	c "github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/player"
)

const (
	HerobrineName = "Herobrine"

	herobrineLifetime  = 20 * 90
	herobrineNearRange = 12.0
	herobrineGazeTicks = 6
	herobrineGazeCos   = 0.97
	playerEyeHeight    = 1.62
)

func NewHerobrine(id, target int32, x, y, z float64, dim int32) *Mob {
	m := NewMob(id, c.Herobrine, x, y, z, dim)
	m.TargetId = target
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

func (m *Mob) herobrineAI(w WorldShared) {
	m.forward, m.strafe, m.jumping = 0, 0, false
	p := validTarget(w, m.TargetId, m.Dimension)
	if p == nil || m.Age > herobrineLifetime {
		m.Vanished = true
		return
	}
	m.faceTowards(p.X, p.Y, p.Z, playerEyeHeight, 4, 4)

	dx, dy, dz := p.X-m.X, p.Y-m.Y, p.Z-m.Z
	if dx*dx+dy*dy+dz*dz < herobrineNearRange*herobrineNearRange {
		m.Vanished = true
		return
	}
	if m.InSightOf(w, p, herobrineGazeCos) {
		m.gaze++
	} else {
		m.gaze = 0
	}
	if m.gaze >= herobrineGazeTicks {
		m.Vanished = true
	}
}
