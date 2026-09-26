package entities

import (
	"math"
	"sync"
	"time"

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

func (et *EntityTracker) ResetViewer(w WorldShared, viewerId int32) {
	et.Mu.Lock()
	defer et.Mu.Unlock()
	pl, ok := w.GetPlayer(viewerId)
	if !ok {
		return
	}
	for eId := range et.visible[viewerId] {
		//log.Printf("Reset: Despawning %d for %s (%d)", eId, pl.Username, viewerId)
		pl.Connection.Write(w.DespawnEntity(eId))
		et.visible[viewerId][eId] = false
	}
	delete(et.visible, viewerId)
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

// SendToViewers writes data only to the players currently tracking the entity
func (et *EntityTracker) SendToViewers(w WorldShared, entityId int32, data []byte) {
	et.Mu.Lock()
	defer et.Mu.Unlock()
	for viewerId, seen := range et.visible {
		if !seen[entityId] {
			continue
		}
		if pl, ok := w.GetPlayer(viewerId); ok && pl.LoggedIn {
			pl.Connection.Write(data)
		}
	}
}

// SendToViewersAndSelf is SendToViewers plus the entity itself when it is a player (e.g. the rider on a mount packet)
func (et *EntityTracker) SendToViewersAndSelf(w WorldShared, entityId int32, data []byte) {
	et.SendToViewers(w, entityId, data)
	if pl, ok := w.GetPlayer(entityId); ok && pl.LoggedIn {
		pl.Connection.Write(data)
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
		droppedItem, _ := target.(*DroppedItem)
		itemGone := droppedItem != nil && droppedItem.Dead
		alive := targetType == c.FallingBlock || (targetType == c.DroppedItem && !itemGone) || target.GetHP() > 0

		ms := target.GetMovementState()

		posAndRotChanged := ms.PosAndRotChanged()
		posChanged := ms.PosChanged()
		rotChanged := ms.RotChanged()
		velChanged := ms.VChanged()
		teleported := ms.Teleported
		isHurt := ms.IsHurt
		isDead := ms.IsDead
		armSwung := ms.ArmSwing
		sneakChanged := ms.SneakChanged
		wentToBed := ms.WentToBed
		gotUp := ms.GotUp
		// Snapshotting the entity info per this tick
		msCopy := *ms

		var playerMove, mobMove, mobMeta []byte
		if t, ok := target.(*player.Player); ok && targetType == c.Player {
			x, y, z := t.GetPosition()
			playerMove = trackMovePacket(w, t, &t.MovementState, x, y, z, t.Yaw, t.Pitch, 1)
		}
		if t, ok := target.(*Mob); ok && alive {
			mobMove = trackMovePacket(w, t, &t.MovementState, t.X, t.Y, t.Z, t.RotYaw, t.RotPitch, mobUpdateEvery)
			mobMeta = t.TakeMetadata()
		}

		for _, viewer := range viewers {
			viewerID := viewer.GetEntityId()
			if viewerID == targetID {
				continue
			}

			// viewerHP := viewer.GetHP()
			// if viewerHP <= 0 {
			// 	et.ResetViewer(w, viewerID)
			// 	continue
			// }

			x1, _, z1 := viewer.GetPosition()
			dx := math.Abs(x1 - x2)
			dz := math.Abs(z1 - z2)

			isVisible := et.visible[viewerID][targetID]
			sameDim := viewer.GetDim() == target.GetDim()
			inRange := sameDim && dx <= distance && dz <= distance

			if isVisible && !alive && !isDead {
				switch targetType {
				case c.Player, c.Mob:
					viewer.Connection.Write(w.NewEntityEventPacket(target, 3))
				}
			}

			if isVisible && alive {
				switch targetType {
				case c.Player:
					t, _ := target.(*player.Player)
					if gotUp {
						viewer.Connection.Write(w.NewAnimationPacket(t, 3))
					}

					if wentToBed {
						// TODO: Figure out a better solution;
						// Goal: interact animation has to be visible before player goes to bed
						viewer.Connection.Write(w.NewAnimationPacket(t, 1))
						go func() {
							time.Sleep(time.Millisecond * 500)
							viewer.Connection.Write(w.NewInteractWithBlockPacket(t, 0))
						}()
					}

					if sneakChanged {
						viewer.Connection.Write(w.NewEntityMetadataPacket(t, t.SneakingMetadata()))
					}

					if isHurt {
						viewer.Connection.Write(w.NewEntityEventPacket(t, 2))
					}

					if armSwung {
						viewer.Connection.Write(w.NewAnimationPacket(t, 1))
					}

					if playerMove != nil {
						viewer.Connection.Write(playerMove)
					}

				case c.Mob:
					t, _ := target.(*Mob)
					if isHurt {
						viewer.Connection.Write(w.NewEntityEventPacket(t, 2))
					}

					if mobMeta != nil {
						viewer.Connection.Write(w.NewEntityMetadataPacket(t, mobMeta))
					}

					if mobMove != nil {
						viewer.Connection.Write(mobMove)
					}
					if velChanged {
						viewer.Connection.Write(w.NewEntityVelocityPacket(t.GetEntityId(), msCopy))
					}

				case c.Ridable:
					t, _ := target.(*RideableEntity)
					if isHurt {
						viewer.Connection.Write(w.NewEntityEventPacket(t, 2))
					}
					if teleported {
						viewer.Connection.Write(w.NewTeleportPacket(t, msCopy))
					}
					if posAndRotChanged {
						viewer.Connection.Write(w.NewPositionAndRotationOrTeleportPacket(t, msCopy))
					}
					if velChanged {
						viewer.Connection.Write(w.NewEntityVelocityPacket(t.GetEntityId(), msCopy))

					}
				case c.DroppedItem:
					t, _ := target.(*DroppedItem)
					if t.CollectorId != -1 {
						collect := w.NewCollectItemPacket(targetID, t.CollectorId)
						viewer.Connection.Write(collect)
						w.RemoveEntity(targetID)
					} else {
						// the client simulates the item itself, this only corrects drift
						if teleported {
							viewer.Connection.Write(w.NewTeleportPacket(t, msCopy))
						}
						if velChanged {
							viewer.Connection.Write(w.NewEntityVelocityPacket(targetID, msCopy))
						}
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
					//log.Printf("Tracker: Spawning %s (%d) for %s (%d)", t.Username, targetID, viewer.Username, viewerID)
					viewer.Connection.Write(w.SpawnPlayerPacket(t))
					w.SetEquipment(t, viewer)

				case c.Ridable, c.FallingBlock:
					viewer.Connection.Write(w.SpawnObjectPacket(target))

				case c.Mob:
					t, _ := target.(*Mob)
					viewer.Connection.Write(w.SpawnMobPacket(t))

				case c.DroppedItem:
					//log.Printf("Tracker: Spawning %d for %s (%d)", targetID, viewer.Username, viewerID)
					viewer.Connection.Write(w.SpawnItemPacket(target))
					// the spawn packet only carries the velocity coarsely (1/128), so follow up with
					// the exact one; items at rest don't need it
					if msCopy.VelocityX != 0 || msCopy.VelocityY != 0 || msCopy.VelocityZ != 0 {
						viewer.Connection.Write(w.NewEntityVelocityPacket(targetID, msCopy))
					}
				}
				et.visible[viewerID][targetID] = true

			} else if isVisible && (!inRange || shouldDespawn(target)) {
				switch targetType {
				case c.Player, c.Mob, c.Ridable, c.FallingBlock, c.DroppedItem:
					if !inRange || !alive || shouldDespawn(target) {
						// if !alive {
						// 	log.Println("Not alive")
						// }
						// if !inRange {
						// 	log.Printf("Out of range (sameDim=%t) dx=%f<=dist && dz=%f<=dist, dist=%d", sameDim, dx, dz, distance)

						// }
						// if shouldDespawn(target) {
						// 	log.Println("Should despawn")
						// }
						//log.Printf("Tracker: Despawning %d for %s (%d)", targetID, viewer.Username, viewerID)
						viewer.Connection.Write(w.DespawnEntity(targetID))
						delete(et.visible[viewerID], targetID)

						if (targetType != c.Player && !alive) || targetType == c.DroppedItem || targetType == c.FallingBlock {
							//log.Println("Removing Entity")
							w.RemoveEntity(targetID)
						}
					}
					continue
				default:
					continue
				}
			}
		}

		if itemGone {
			w.RemoveEntity(targetID)
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
		if !alive {
			ms.IsDead = true
		}
		if sneakChanged {
			ms.SneakChanged = false
		}
		if wentToBed {
			ms.WentToBed = false
		}
		if gotUp {
			ms.GotUp = false
		}
	}
}

const (
	playerMinPosDelta     = 2  // 1/16 block
	playerMinRotDelta     = 2  // ~3 degrees
	playerForceTeleportIn = 40 // resync every 2s so rounding drift can't build up
	mobUpdateEvery        = 3
)

func quantizeAngle(deg float32) int32 {
	return int32(math.Floor(float64(deg)*256/360)) & 0xFF
}

func trackMovePacket(w WorldShared, e c.Entity, ms *c.MovementState, x, y, z float64, yaw, pitch float32, every int) []byte {
	ms.TicksSinceTeleport++
	ms.UpdateCounter++
	if ms.EncInit && ms.UpdateCounter < every && ms.TicksSinceTeleport < playerForceTeleportIn {
		return nil
	}
	ms.UpdateCounter = 0

	qx, qy, qz := int32(math.Floor(x*32)), int32(math.Floor(y*32)), int32(math.Floor(z*32))
	qYaw, qPitch := quantizeAngle(yaw), quantizeAngle(pitch)

	m := c.MovementState{
		X: x, Y: y, Z: z, Yaw: yaw, Pitch: pitch,
		PrevX: float64(ms.EncX) / 32, PrevY: float64(ms.EncY) / 32, PrevZ: float64(ms.EncZ) / 32,
	}
	dx, dy, dz := qx-ms.EncX, qy-ms.EncY, qz-ms.EncZ
	rotDelta := func(a, b int32) int32 {
		d := (a - b) & 0xFF
		if d > 128 {
			d = 256 - d
		}
		return d
	}
	needsRot := rotDelta(qYaw, ms.EncYaw) >= playerMinRotDelta || rotDelta(qPitch, ms.EncPitch) >= playerMinRotDelta
	needsMove := abs32(dx) >= playerMinPosDelta || abs32(dy) >= playerMinPosDelta || abs32(dz) >= playerMinPosDelta

	var pkt []byte
	switch {
	case !ms.EncInit || ms.TicksSinceTeleport >= playerForceTeleportIn ||
		dx < -128 || dx > 127 || dy < -128 || dy > 127 || dz < -128 || dz > 127:
		pkt = w.NewTeleportPacket(e, m)
		ms.TicksSinceTeleport = 0
	case needsMove && needsRot:
		pkt = w.NewPositionAndRotationOrTeleportPacket(e, m)
	case needsMove:
		pkt = w.NewPositionPacket(e, m)
		qYaw, qPitch = ms.EncYaw, ms.EncPitch
	case needsRot:
		pkt = w.NewRotationPacket(e, m)
		qx, qy, qz = ms.EncX, ms.EncY, ms.EncZ
	default:
		return nil
	}
	ms.EncX, ms.EncY, ms.EncZ = qx, qy, qz
	ms.EncYaw, ms.EncPitch = qYaw, qPitch
	ms.EncInit = true
	return pkt
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
