package entities

import (
	"math"
	"sync"

	c "github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/player"
)

const VIEW_DISTANCE = 12

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

func (et *EntityTracker) Manage(w WorldShared) {
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

	viewers := make([]*player.Player, 0, len(w.GetPlayers()))
	for _, viewer := range w.GetPlayers() {
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

	for _, target := range w.SnapshotEntities() {
		targetID := target.GetEntityId()
		targetType := target.GetEntityType()

		if targetType == c.Player && !target.GetLoggedIn() {
			continue
		}

		x2, _, z2 := target.GetPosition()
		alive := targetType == c.FallingBlock || targetType == c.DroppedItem || target.GetHP() > 0

		ms := target.GetMovementState()

		posAndRotChanged := ms.PosAndRotChanged()
		posChanged := ms.PosChanged()
		rotChanged := ms.RotChanged()
		velChanged := ms.VChanged()
		teleported := ms.Teleported
		isHurt := ms.IsHurt
		armSwung := ms.ArmSwing
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

			if isVisible && !alive {
				switch targetType {
				case c.Player, c.Mob:
					viewer.Connection.Write(w.NewEntityEventPacket(target, 3))
				}
			}

			if isVisible && alive {
				switch targetType {
				case c.Player:
					t, _ := target.(*player.Player)
					if isHurt {
						viewer.Connection.Write(w.NewEntityEventPacket(t, 2))
					}

					if armSwung {
						viewer.Connection.Write(w.NewAnimationPacket(t, 1))
					}

					if posAndRotChanged || posChanged || velChanged || rotChanged {
						viewer.Connection.Write(w.NewPositionAndRotationOrTeleportPacket(t, msCopy))
						viewer.Connection.Write(w.NewPositionPacket(t, msCopy))
						viewer.Connection.Write(w.NewEntityVelocityPacket(t.GetEntityId(), msCopy))
						viewer.Connection.Write(w.NewRotationPacket(t, msCopy))
					}

				case c.Mob:
					t, _ := target.(*Mob)
					if isHurt {
						viewer.Connection.Write(w.NewEntityEventPacket(t, 2))
					}

					if posAndRotChanged {
						viewer.Connection.Write(w.NewMobPositionAndRotationOrTeleportPacket(t, msCopy))
					}
					if velChanged {
						viewer.Connection.Write(w.NewEntityVelocityPacket(t.GetEntityId(), msCopy))
					}

				case c.Ridable:
					t, _ := target.(*RideableEntity)
					if isHurt {
						viewer.Connection.Write(w.NewEntityEventPacket(t, 2))
					}
					if posAndRotChanged || velChanged || teleported {
						viewer.Connection.Write(w.NewPositionAndRotationOrTeleportPacket(t, msCopy))
						viewer.Connection.Write(w.NewEntityVelocityPacket(t.GetEntityId(), msCopy))
						viewer.Connection.Write(w.NewTeleportPacket(t, msCopy))

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
					viewer.Connection.Write(w.SpawnPlayerPacket(t))
					w.SetEquipment(t, viewer)

				case c.Ridable, c.FallingBlock:
					viewer.Connection.Write(w.SpawnObjectPacket(target))

				case c.Mob:
					t, _ := target.(*Mob)
					viewer.Connection.Write(w.SpawnMobPacket(t))

				case c.DroppedItem:
					viewer.Connection.Write(w.SpawnItemPacket(target))
					if velChanged {
						viewer.Connection.Write(w.NewEntityVelocityPacket(targetID, msCopy))
					}
				}
				et.visible[viewerID][targetID] = true

			} else if isVisible && (!inRange || (isDespawnable(targetType) && shouldDespawn(target))) {
				switch targetType {
				case c.Player, c.Mob, c.Ridable, c.FallingBlock:
					if !inRange || !alive || shouldDespawn(target) {
						viewer.Connection.Write(w.DespawnEntity(targetID))
						delete(et.visible[viewerID], targetID)

						if targetType != c.Player {
							w.RemoveEntity(targetID)
						}
					}
					continue
				default:
					continue
				}
			}
		}

		// Notify server that information has been sent to clients
		// TODO: For players this still seems to be a bit jank though...
		if posAndRotChanged {
			ms.PositionAndRotationChanged = false
		}
		if velChanged {
			ms.VelocityChanged = false
		}
		if teleported {
			ms.Teleported = false
		}
		if rotChanged {
			ms.RotationChanged = false
		}
		if posChanged {
			ms.PositionChanged = false
		}
		if isHurt {
			ms.IsHurt = false
		}
		if armSwung {
			ms.ArmSwing = false
		}
	}
}
