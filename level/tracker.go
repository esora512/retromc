package level

import (
	"math"
	"sync"

	c "github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/player"
)

type EntityTracker struct {
	visible map[int32]map[int32]bool
	Mu      sync.Mutex
}

func NewEntityTracker() *EntityTracker {
	return &EntityTracker{
		visible: make(map[int32]map[int32]bool),
	}
}

// Use case: Server has to resend entity packets if the viewer died & respawned, so client has them again
func (et *EntityTracker) ResetViewer(playerID int32) {
	et.Mu.Lock()
	defer et.Mu.Unlock()
	delete(et.visible, playerID)
}

// Clears the entity server side, so if it is still present in w.Entities, it gets re-spawned
func (et *EntityTracker) ResetEntity(id int32) {
	et.Mu.Lock()
	defer et.Mu.Unlock()
	delete(et.visible, id)
	for _, seen := range et.visible {
		delete(seen, id)
	}
}

// func (et *EntityTracker) Add(playerId int32, otherId int32) {
// 	et.Mu.Lock()
// 	defer et.Mu.Unlock()
// 	if et.visible[playerId] == nil {
// 		et.visible[playerId] = make(map[int32]bool)
// 	}
// 	et.visible[playerId][otherId] = true
// }

// func (et *EntityTracker) AddForAll(w *World, otherId int32) {
// 	et.Mu.Lock()
// 	defer et.Mu.Unlock()
// 	for p := range w.Players {
// 		if et.visible[p] == nil {
// 			et.visible[p] = make(map[int32]bool)
// 		}
// 		et.visible[p][otherId] = true
// 	}
// }

func (et *EntityTracker) Despawn(w *World, id int32) {
	et.Mu.Lock()
	delete(et.visible, id)
	for _, seen := range et.visible {
		delete(seen, id)
	}
	et.Mu.Unlock()

	w.BroadcastPacket(w.despawnEntity(id))
}

type Entity interface {
	GetName() string
	GetPosition() (float64, float64, float64)
	SetPosition(x, y, z float64)
	GetEntityId() int32
	SetHP(hp int16)
	GetHP() int16
	GetLoggedIn() bool
	GetDim() int32
	GetVelocity() (float64, float64, float64)
	Despawn() bool
	GetMovementState() *c.MovementState
	GetEntityType() c.EntityType
}

func (et *EntityTracker) Manage(w *World) {
	et.Mu.Lock()
	defer et.Mu.Unlock()
	const distance = VIEW_DISTANCE * 8

	despawnResults := make(map[int32]bool)
	shouldDespawn := func(target Entity) bool {
		id := target.GetEntityId()
		if res, ok := despawnResults[id]; ok {
			return res
		}
		res := target.Despawn()
		despawnResults[id] = res
		return res
	}

	sendVelocityIfChanged := func(viewer *player.Player, target Entity, ms *c.MovementState) {
		if ms.VelocityChanged {
			p := w.newEntityVelocityPacket(target.GetEntityId(), ms.VelocityX, ms.VelocityY, ms.VelocityZ)
			viewer.Connection.Write(p)
			ms.VelocityChanged = false
		}
	}

	isDespawnable := func(t c.EntityType) bool {
		switch t {
		case c.FallingBlock, c.Mob, c.Player, c.Ridable:
			return true
		default:
			return false
		}
	}

	for _, viewer := range w.Players {
		viewerID := viewer.GetEntityId()

		if et.visible[viewerID] == nil {
			et.visible[viewerID] = make(map[int32]bool)
		}

		if !viewer.LoggedIn {
			continue
		}

		if viewer.GetEntityType() == c.Player && viewer.HP <= 0 {
			continue
		}

		x1, _, z1 := viewer.GetPosition()

		for _, target := range w.Entities {
			targetID := target.GetEntityId()
			targetType := target.GetEntityType()

			if viewerID == targetID {
				continue
			}

			if targetType == c.Player && !target.GetLoggedIn() {
				continue
			}

			x2, _, z2 := target.GetPosition()

			dx := math.Abs(x1 - x2)
			dz := math.Abs(z1 - z2)

			isVisible := et.visible[viewerID][targetID]
			sameDim := viewer.GetDim() == target.GetDim()
			inRange := sameDim && dx <= distance && dz <= distance
			alive := targetType == c.FallingBlock || targetType == c.DroppedItem || target.GetHP() > 0

			if isVisible && alive {
				ms := target.GetMovementState()

				switch targetType {
				case c.Mob:
					t, _ := target.(*Mob)
					if ms.PositionAndRotationChanged {
						p := w.newMobPositionAndRotationOrTeleportPacket(t, *ms)
						viewer.Connection.Write(p)
						ms.PositionAndRotationChanged = false
					}

				// case c.Player:
				// 	t, _ := target.(*player.Player)
				// 	if ms.VelocityChanged {
				// 		p := w.newEntityVelocityPacket(targetID, ms.VelocityX, ms.VelocityY, ms.VelocityZ)
				// 		viewer.Connection.Write(p)
				// 		ms.VelocityChanged = false
				// 	}
				// 	if ms.PositionAndRotationChanged {
				// 		p := w.newPositionAndRotationOrTeleportPacket(w, t, *ms)
				// 		viewer.Connection.Write(p)
				// 		ms.PositionAndRotationChanged = false
				// 	}
				// 	if ms.PositionChanged {
				// 		p := w.newPositionPacket(w, t, *ms)
				// 		viewer.Connection.Write(p)
				// 		ms.PositionChanged = false
				// 	}
				// 	if ms.RotationChanged {
				// 		p := w.newPositionPacket(w, t, *ms)
				// 		viewer.Connection.Write(p)
				// 		ms.RotationChanged = false
				// 	}

				case c.Ridable:
					t, _ := target.(*entities.RideableEntity)
					if ms.PositionAndRotationChanged {
						p := w.newPositionAndRotationOrTeleportPacket(w, t, *ms)
						viewer.Connection.Write(p)
						ms.PositionAndRotationChanged = false
					}
					sendVelocityIfChanged(viewer, target, ms)
					if ms.Teleported {
						p := w.newTeleportPacket(w, t, *ms)
						viewer.Connection.Write(p)
						ms.Teleported = false
					}

				case c.DroppedItem, c.FallingBlock:
					sendVelocityIfChanged(viewer, target, ms)
				}
			}

			if !isVisible && inRange && alive {
				switch targetType {
				case c.Player:
					if target.GetName() == viewer.Username {
						continue
					}
					t, _ := target.(*player.Player)
					viewer.Connection.Write(w.spawnPlayer(t))
					w.setEquipment(t, viewer.Connection.Write)

				case c.Ridable:
					viewer.Connection.Write(w.spawnObject(target))

				case c.Mob:
					t, _ := target.(*Mob)
					viewer.Connection.Write(w.spawnMob(t))

				case c.DroppedItem:
					t, _ := target.(*DroppedItem)
					viewer.Connection.Write(w.spawnItem(t))
					sendVelocityIfChanged(viewer, target, target.GetMovementState())
				}
				et.visible[viewerID][targetID] = true

			} else if isVisible && (!inRange || (isDespawnable(targetType) && shouldDespawn(target))) {
				switch targetType {
				case c.Player, c.Mob, c.Ridable, c.FallingBlock:
					despawn := !inRange || !alive || shouldDespawn(target)
					if despawn {
						viewer.Connection.Write(w.despawnEntity(targetID))
						delete(et.visible[viewerID], targetID)

						if targetType == c.Mob || targetType == c.Ridable || targetType == c.FallingBlock {
							w.RemoveEntity(targetID)
						}
					}
					continue

				default:
					continue
					// viewer.Connection.Write(et.DespawnEntity(targetID))
					// delete(et.visible[viewerID], targetID)
					// w.RemoveEntity(targetID)
				}
			}
		}
	}
}
