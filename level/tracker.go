package level

import (
	"log"
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

func (et *EntityTracker) Despawn(w *World, id int32) {
	et.Mu.Lock()
	delete(et.visible, id)
	for _, seen := range et.visible {
		delete(seen, id)
	}
	et.Mu.Unlock()

	w.BroadcastPacket(w.despawnEntity(id))
}

func (et *EntityTracker) Manage(w *World) {
	et.Mu.Lock()
	defer et.Mu.Unlock()
	const distance = VIEW_DISTANCE * 8

	despawnResults := make(map[int32]bool)
	shouldDespawn := func(target c.Entity) bool {
		id := target.GetEntityId()
		if res, ok := despawnResults[id]; ok {
			return res
		}
		res := target.Despawn()
		despawnResults[id] = res
		return res
	}

	isDespawnable := func(t c.EntityType) bool {
		switch t {
		case c.FallingBlock, c.Mob, c.Player, c.Ridable:
			return true
		default:
			return false
		}
	}

	viewers := make([]*player.Player, 0, len(w.Players))
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
		viewers = append(viewers, viewer)
	}

	for _, target := range w.Entities {
		targetID := target.GetEntityId()
		targetType := target.GetEntityType()

		if targetType == c.Player && !target.GetLoggedIn() {
			continue
		}

		x2, _, z2 := target.GetPosition()
		alive := targetType == c.FallingBlock || targetType == c.DroppedItem || target.GetHP() > 0

		ms := target.GetMovementState()

		posAndRotChanged := ms.PositionAndRotationChanged
		posChanged := ms.PositionChanged
		rotChanged := ms.RotationChanged
		velChanged := ms.VelocityChanged
		teleported := ms.Teleported
		// Snapshotting the entity info per this tick
		msCopy := *ms

		for _, viewer := range viewers {
			viewerID := viewer.GetEntityId()
			if viewerID == targetID {
				continue
			}

			x1, _, z1 := viewer.GetPosition()
			dx := math.Abs(x1 - x2)
			dz := math.Abs(z1 - z2)

			isVisible := et.visible[viewerID][targetID]
			sameDim := viewer.GetDim() == target.GetDim()
			inRange := sameDim && dx <= distance && dz <= distance

			if isVisible && alive {
				switch targetType {
				case c.Player:
					t, _ := target.(*player.Player)
					if posAndRotChanged {
						viewer.Connection.Write(w.newPositionAndRotationOrTeleportPacket(w, t, msCopy))
					}
					if posChanged {
						viewer.Connection.Write(w.newPositionPacket(w, t, msCopy))
					}
					if velChanged {
						viewer.Connection.Write(w.newEntityVelocityPacket(t.GetEntityId(), msCopy))
					}
					if rotChanged {
						viewer.Connection.Write(w.newRotationPacket(w, t, msCopy))
					}

				case c.Mob:
					t, _ := target.(*entities.Mob)
					if posAndRotChanged {
						viewer.Connection.Write(w.newMobPositionAndRotationOrTeleportPacket(t, msCopy))
					}
					if velChanged {
						viewer.Connection.Write(w.newEntityVelocityPacket(t.GetEntityId(), msCopy))
					}

				case c.Ridable:
					t, _ := target.(*entities.RideableEntity)
					if posAndRotChanged {
						viewer.Connection.Write(w.newPositionAndRotationOrTeleportPacket(w, t, msCopy))
					}
					if velChanged {
						viewer.Connection.Write(w.newEntityVelocityPacket(t.GetEntityId(), msCopy))
					}
					if teleported {
						viewer.Connection.Write(w.newTeleportPacket(w, t, msCopy))
					}

				case c.FallingBlock:
					if velChanged {
						viewer.Connection.Write(w.newEntityVelocityPacket(targetID, msCopy))
					}
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
					t, _ := target.(*entities.Mob)
					log.Printf("Spawning Spider for %s", viewer.Username)
					viewer.Connection.Write(w.spawnMob(t))

				case c.DroppedItem:
					log.Printf("Spawning Drop...")
					t, _ := target.(*DroppedItem)
					viewer.Connection.Write(w.spawnItem(t))
					if velChanged {
						viewer.Connection.Write(w.newEntityVelocityPacket(targetID, msCopy))
					}
				}
				et.visible[viewerID][targetID] = true

			} else if isVisible && (!inRange || (isDespawnable(targetType) && shouldDespawn(target))) {
				switch targetType {
				case c.Player, c.Mob, c.Ridable, c.FallingBlock:
					log.Printf("Despawning because: inRange=%t, alive=%t", inRange, alive)
					if !inRange || !alive || shouldDespawn(target) {
						if targetType == c.Mob {
							log.Printf("Despawning Spider for %s", viewer.Username)
						}
						viewer.Connection.Write(w.despawnEntity(targetID))
						delete(et.visible[viewerID], targetID)
					}
					continue
				default:
					continue
				}
			}
		}

		// Notify server that information has been sent to clients
		// TODO: Atm, we disable this since it leads to jank AF client rendering; need to fix this at some point
		// if posAndRotChanged {
		// 	ms.PositionAndRotationChanged = false
		// }
		// if velChanged {
		// 	ms.VelocityChanged = false
		// }
		// if teleported {
		// 	ms.Teleported = false
		// }
		// if rotChanged {
		// 	ms.RotationChanged = false
		// }
		// if posChanged {
		// 	ms.PositionChanged = false
		// }
	}
}
