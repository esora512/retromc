package level

import (
	"math"
	"math/rand"

	c "github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	e "github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/player"
)

func NewSpider(w *World, x, y, z float64, dim int32) *e.Mob {
	return e.NewMob(w.NextEntityId(), c.Spider, x, y, z, dim)
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

func (w *World) SpawnMobType(mobType byte, x, y, z, dim int32, target int32) int32 {
	m := e.NewMob(w.NextEntityId(), mobType, float64(x)+0.5, float64(y), float64(z)+0.5, dim)
	m.SetTarget(target)
	w.Entities[m.EntityId] = m
	return m.EntityId
}

func (w *World) SpawnSpider(x, y, z, dim int32, target int32) int32 {
	return w.SpawnMobType(c.Spider, x, y, z, dim, target)
}

func NewSkeleton(w *World, x, y, z float64, dim int32) *e.Mob {
	return e.NewMob(w.NextEntityId(), c.Skeleton, x, y, z, dim)
}

func (w *World) SpawnSkeleton(x, y, z, dim int32, target int32) int32 {
	return w.SpawnMobType(c.Skeleton, x, y, z, dim, target)
}

func NewPig(w *World, x, y, z float64, dim int32) *e.Mob {
	return e.NewMob(w.NextEntityId(), c.Pig, x, y, z, dim)
}

func (w *World) SpawnPig(x, y, z, dim int32) int32 {
	return w.SpawnMobType(c.Pig, x, y, z, dim, -1)
}

var hostileTypes = []byte{c.Zombie, c.Skeleton, c.Spider, c.Creeper}
var animalTypes = []byte{c.Pig, c.Sheep}

func randomType(types []byte) byte {
	return types[rand.Intn(len(types))]
}

func (w *World) Herobrine(target int32) (*e.Mob, bool) {
	for _, ent := range w.Entities {
		if m, ok := ent.(*e.Mob); ok && m.IsHerobrine() && m.TargetId == target {
			return m, true
		}
	}
	return nil, false
}

// SpawnHerobrine places him at a distance, out of the player's view but with a clear line of sight
func (w *World) SpawnHerobrine(pl *player.Player) (int32, bool) {
	if _, ok := w.Herobrine(pl.GetEntityId()); ok {
		return 0, false
	}
	for range 24 {
		sx, sz := randomPointOnRing(pl.X, pl.Z, 26)
		x, z := int32(math.Floor(sx)), int32(math.Floor(sz))
		y, ok := w.findGroundY(x, z, int32(pl.Y), pl.Dimension)
		if !ok {
			continue
		}
		m := e.NewHerobrine(w.NextEntityId(), pl.GetEntityId(), float64(x)+0.5, float64(y), float64(z)+0.5, pl.Dimension)
		if m.InSightOf(w, pl, 0.5) || !m.InSightOf(w, pl, -1) {
			continue
		}
		w.Entities[m.EntityId] = m
		return m.EntityId, true
	}
	return 0, false
}

func (w *World) GroundY(x, z, startY, dim int32) (int32, bool) {
	return w.findGroundY(x, z, startY, dim)
}
