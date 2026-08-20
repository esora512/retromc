package level

import (
	"math"
	"sync"

	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/player"
)

type EntityTracker struct {
	SpawnPlayer   func(pl *player.Player) []byte
	SpawnObject   func(e Entity) []byte
	SpawnMob      func(m *Mob) []byte
	DespawnEntity func(id int32) []byte
	SpawnItem     func(d *DroppedItem) []byte
	SetEquipment  func(pl *player.Player, send func([]byte) (int, error))
	visible       map[int32]map[int32]bool
	Mu            sync.Mutex
}

func NewEntityTracker(
	spawnPlayer func(pl *player.Player) []byte,
	spawnObject func(e Entity) []byte,
	spawnMob func(m *Mob) []byte,
	spawnItem func(d *DroppedItem) []byte,
	despawnEntity func(id int32) []byte,
	setEquipment func(pl *player.Player, send func([]byte) (int, error)),
) *EntityTracker {
	return &EntityTracker{
		SpawnPlayer:   spawnPlayer,
		SpawnObject:   spawnObject,
		SpawnMob:      spawnMob,
		SpawnItem:     spawnItem,
		DespawnEntity: despawnEntity,
		SetEquipment:  setEquipment,
		visible:       make(map[int32]map[int32]bool),
	}
}

func (et *EntityTracker) ResetViewer(playerID int32) {
	et.Mu.Lock()
	defer et.Mu.Unlock()
	delete(et.visible, playerID)
}

func (et *EntityTracker) ResetEntity(id int32) {
	et.Mu.Lock()
	defer et.Mu.Unlock()
	delete(et.visible, id)
	for _, seen := range et.visible {
		delete(seen, id)
	}
}

func (et *EntityTracker) Add(playerId int32, otherId int32) {
	et.Mu.Lock()
	defer et.Mu.Unlock()
	if et.visible[playerId] == nil {
		et.visible[playerId] = make(map[int32]bool)
	}
	et.visible[playerId][otherId] = true
}

func (et *EntityTracker) AddForAll(w *World, otherId int32) {
	et.Mu.Lock()
	defer et.Mu.Unlock()
	for p := range w.Players {
		if et.visible[p] == nil {
			et.visible[p] = make(map[int32]bool)
		}
		et.visible[p][otherId] = true
	}
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

	for _, viewer := range w.Players {
		viewerID := viewer.GetEntityId()

		if et.visible[viewerID] == nil {
			et.visible[viewerID] = make(map[int32]bool)
		}

		if !viewer.LoggedIn {
			continue
		}

		if viewer.IsPlayer() && viewer.HP <= 0 {
			continue
		}

		x1, _, z1 := viewer.GetPosition()

		for _, target := range w.Entities {
			targetID := target.GetEntityId()

			if viewerID == targetID {
				continue
			}

			if target.IsPlayer() && !target.GetLoggedIn() {
				continue
			}

			x2, _, z2 := target.GetPosition()

			dx := math.Abs(x1 - x2)
			dz := math.Abs(z1 - z2)

			isVisible := et.visible[viewerID][targetID]
			sameDim := viewer.GetDim() == target.GetDim()
			inRange := sameDim && dx <= distance && dz <= distance
			alive := target.IsBlock() || target.IsItem() || target.GetHP() > 0

			if isVisible && alive {
				// if target.IsPlayer() {
				// 	t, _ := target.(*player.Player)
				// 	if t.MovementState.VelocityChanged {
				// 		p := w.newEntityVelocityPacket(targetID, t.MovementState.VelocityX, t.MovementState.VelocityY, t.MovementState.VelocityZ)
				// 		viewer.Connection.Write(p)
				// 		t.MovementState.VelocityChanged = false 
				// 	}

				// 	if t.MovementState.PositionAndRotationChanged {
				// 		p := w.newPositionAndRotationOrTeleportPacket(w, t, t.MovementState)
				// 		viewer.Connection.Write(p)
				// 		t.MovementState.PositionAndRotationChanged = false
				// 	}

				// 	if t.MovementState.PositionChanged {
				// 		p := w.newPositionPacket(w, t, t.MovementState)
				// 		viewer.Connection.Write(p)
				// 		t.MovementState.PositionChanged = false 
				// 	}

				// 	if t.MovementState.RotationChanged {
				// 		p := w.newPositionPacket(w, t, t.MovementState)
				// 		viewer.Connection.Write(p)
				// 		t.MovementState.RotationChanged = false
				// 	}
				// }

				if target.IsItem() {
					t, _ := target.(*DroppedItem)
					if t.MovementState.VelocityChanged {
						p := w.newEntityVelocityPacket(t.EntityId, t.MovementState.VelocityX, t.MovementState.VelocityY, t.MovementState.VelocityZ)
						viewer.Connection.Write(p)
						t.MovementState.VelocityChanged = false
					}
				}

				if target.IsBlock() {
					t, _ := target.(*entities.BlockEntity)
					if t.MovementState.VelocityChanged {
						p := w.newEntityVelocityPacket(t.EntityId, t.MovementState.VelocityX, t.MovementState.VelocityY, t.MovementState.VelocityZ)
						viewer.Connection.Write(p)
						t.MovementState.VelocityChanged = false
					}
				}

				if target.IsRideable() {
					t, _ := target.(*entities.RideableEntity)
					if t.MovementState.PositionAndRotationChanged {
						p := w.newPositionAndRotationOrTeleportPacket(w, t, t.MovementState)
						viewer.Connection.Write(p)
						t.MovementState.PositionAndRotationChanged = false
					}

					if t.MovementState.VelocityChanged {
						p := w.newEntityVelocityPacket(t.EntityId, t.MovementState.VelocityX, t.MovementState.VelocityY, t.MovementState.VelocityZ)
						viewer.Connection.Write(p)
						t.MovementState.VelocityChanged = false
					}

					if t.MovementState.Teleported {
						p := w.newTeleportPacket(w, t, t.MovementState)
						viewer.Connection.Write(p)
						t.MovementState.Teleported = false
					}
				}
			}

			if !isVisible && inRange && alive {
				if target.IsPlayer() {
					if target.GetName() == viewer.Username {
						continue
					}
					t, _ := target.(*player.Player)
					viewer.Connection.Write(et.SpawnPlayer(t))
					et.SetEquipment(t, viewer.Connection.Write)
				} else if target.IsRideable() {
					viewer.Connection.Write(et.SpawnObject(target))
				} else if target.IsMob() {
					t, _ := target.(*Mob)
					viewer.Connection.Write(et.SpawnMob(t))
				} else if target.IsItem() {
					t, _ := target.(*DroppedItem)
					viewer.Connection.Write(et.SpawnItem(t))
					if t.MovementState.VelocityChanged {
						p := w.newEntityVelocityPacket(t.EntityId, t.MovementState.VelocityX, t.MovementState.VelocityY, t.MovementState.VelocityZ)
						viewer.Connection.Write(p)
						t.MovementState.VelocityChanged = false
					}
				}
				et.visible[viewerID][targetID] = true
			} else if isVisible && (!inRange || !alive || (target.IsBlock() && shouldDespawn(target))) {
				if target.IsPlayer() || target.IsMob() || target.IsRideable() || target.IsBlock() {
					despawn := !inRange || !alive || shouldDespawn(target)
					if despawn {
						viewer.Connection.Write(et.DespawnEntity(targetID))
						delete(et.visible[viewerID], targetID)

						if target.IsMob() || target.IsRideable() || target.IsBlock() {
							w.RemoveEntity(targetID)
						}
					}
					continue
				} else {
					viewer.Connection.Write(et.DespawnEntity(targetID))
					delete(et.visible[viewerID], targetID)
					w.RemoveEntity(targetID)
				}
			}
		}
	}
}

func (et *EntityTracker) Despawn(w *World, id int32) {
	et.Mu.Lock()
	delete(et.visible, id)
	for _, seen := range et.visible {
		delete(seen, id)
	}
	et.Mu.Unlock()

	w.BroadcastPacket(et.DespawnEntity(id))
}

type Entity interface {
	GetName() string
	GetPosition() (float64, float64, float64)
	SetPosition(x, y, z float64)
	IsRideable() bool
	GetEntityId() int32
	IsPlayer() bool
	SetHP(hp int16)
	GetHP() int16
	GetLoggedIn() bool
	GetDim() int32
	IsMob() bool
	GetVelocity() (float64, float64, float64)
	IsItem() bool
	Despawn() bool
	IsBlock() bool
}
