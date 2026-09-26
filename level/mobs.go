package level

import (
	"math"
	"math/rand"

	c "github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	e "github.com/leNicDev/retromc/entities"
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
