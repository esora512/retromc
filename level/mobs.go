package level

import (
	"math"

	c "github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	e "github.com/leNicDev/retromc/entities"
)

func NewSpider(w *World, x, y, z float64, dim int32) *e.Mob {
	m := e.Mob{EntityId: w.NextEntityId(),
		X: x, Y: y, Z: z,
		Yaw: 0, Pitch: 0,
		Vx: 0, Vy: 0, Vz: 0,
		Dimension: dim,
		MobType:   c.Spider,
		Metadata:  0,
		TargetId:  -1,
		HP:        10,
		OnGround:  true,
		DespawnIn: -1,
	}
	m.MovementState.IsDead = false
	return &m
}

func (w *World) FindNearbyPlayer(m *entities.Mob) (int32, bool) {
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

func (w *World) SpawnSpider(x, y, z, dim int32, target int32) int32 {
	s := NewSpider(w, float64(x), float64(y), float64(z), dim)
	s.SetTarget(target)
	w.Entities[s.EntityId] = s
	return s.EntityId
}

func NewSkeleton(w *World, x, y, z float64, dim int32) *e.Mob {
	m := e.Mob{EntityId: w.NextEntityId(),
		X: x, Y: y, Z: z,
		Yaw: 0, Pitch: 0,
		Vx: 0, Vy: 0, Vz: 0,
		Dimension: dim,
		MobType:   c.Skeleton,
		Metadata:  0,
		TargetId:  -1,
		HP:        10,
		OnGround:  true,
		DespawnIn: -1,
	}
	m.MovementState.IsDead = false
	return &m
}

func (w *World) SpawnSkeleton(x, y, z, dim int32, target int32) int32 {
	s := NewSkeleton(w, float64(x), float64(y), float64(z), dim)
	s.SetTarget(target)
	w.Entities[s.EntityId] = s
	return s.EntityId
}

func NewPig(w *World, x, y, z float64, dim int32) *e.Mob {
	m := e.Mob{EntityId: w.NextEntityId(),
		X: x, Y: y, Z: z,
		Yaw: 0, Pitch: 0,
		Vx: 0, Vy: 0, Vz: 0,
		Dimension: dim,
		MobType:   c.Pig,
		Metadata:  0,
		TargetId:  -1,
		HP:        10,
		OnGround:  true,
		DespawnIn: -1,
	}
	m.MovementState.IsDead = false
	return &m
}

func (w *World) SpawnPig(x, y, z, dim int32) int32 {
	p := NewPig(w, float64(x), float64(y), float64(z), dim)
	w.Entities[p.EntityId] = p
	return p.EntityId
}
