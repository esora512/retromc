package level

import (
	"log"
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
			alive := target.GetHP() > 0

			if isVisible && alive {
				if target.IsBlock() {
					t, _ := target.(*entities.BlockEntity)
					if t.MovementState.VelocityChanged {
						w.BroadcastEntityVelocity(
							t.EntityId,
							t.MovementState.VelocityX,
							t.MovementState.VelocityY,
							t.MovementState.VelocityZ,
						)
						t.MovementState.VelocityChanged = false
					}
				}

				if target.IsRideable() {
					t, _ := target.(*entities.RideableEntity)
					if t.MovementState.PositionChanged {
						w.BroadcastPositionAndRotation(
							t,
							t.MovementState.PrevX,
							t.MovementState.PrevY,
							t.MovementState.PrevZ,
							t.MovementState.X,
							t.MovementState.Y,
							t.MovementState.Z,
							t.Yaw,
						)
						t.MovementState.PositionChanged = false
					}

					if t.MovementState.VelocityChanged {
						w.BroadcastEntityVelocity(
							t.EntityId,
							t.MovementState.VelocityX,
							t.MovementState.VelocityY,
							t.MovementState.VelocityZ,
						)
						t.MovementState.VelocityChanged = false
					}

					if t.MovementState.Teleported {
						w.BroadcastTeleport(
							t,
							t.MovementState.X,
							t.MovementState.Y,
							t.MovementState.Z,
							t.Yaw)
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
					log.Printf("Spawning %s for %s", target.GetName(), viewer.GetName())
				} else if target.IsRideable() {
					viewer.Connection.Write(et.SpawnObject(target))
				} else if target.IsMob() {
					t, _ := target.(*Mob)
					viewer.Connection.Write(et.SpawnMob(t))
				} else if target.IsItem() {
					t, _ := target.(*DroppedItem)
					viewer.Connection.Write(et.SpawnItem(t))
				}
				et.visible[viewerID][targetID] = true
			} else if isVisible && (!inRange || !alive) {
				if target.IsPlayer() || target.IsMob() || target.IsRideable() || target.IsBlock() {
					despawn := target.Despawn()
					if despawn {
						log.Println("Despawning Living Entity")
						viewer.Connection.Write(et.DespawnEntity(targetID))
						delete(et.visible[viewerID], targetID)

						if target.IsMob() || target.IsRideable() || target.IsBlock() {
							w.RemoveEntity(targetID)
						}
					}
					return
				} else {
					viewer.Connection.Write(et.DespawnEntity(targetID))
					delete(et.visible[viewerID], targetID)
					log.Printf("Despawning %s for %s", target.GetName(), viewer.GetName())
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
