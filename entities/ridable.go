package entities

import (
	"fmt"
	"math"

	"github.com/leNicDev/retromc/constants"
)

type RideableEntity struct {
	EntityId      int32
	X             float64
	Y             float64
	Z             float64
	VelocityX     float64
	VelocityY     float64
	VelocityZ     float64
	LastSentVelX  float64
	LastSentVelY  float64
	LastSentVelZ  float64
	OwnerEntityId int32
	ObjectType    byte

	PassengerEntityId  int32
	PassengerVelocityX float64
	PassengerVelocityZ float64
	Yaw                byte
	YawDegrees         float64

	HP        int16
	Dimension int32

	ShouldDespawn bool

	MovementState constants.MovementState
}

func (r *RideableEntity) SetTeleportMovement(nextX, nextY, nextZ float64, yaw byte) {
	r.MovementState.X = nextX
	r.MovementState.Y = nextY
	r.MovementState.Z = nextZ
	r.Yaw = yaw
	r.MovementState.Teleported = true 
}

func (r *RideableEntity) SetVelocityMovement(vx, vy, vz float64) {
	r.MovementState.VelocityX = vx
	r.MovementState.VelocityY = vy
	r.MovementState.VelocityZ = vz
	r.MovementState.VelocityChanged = true
}

func (r *RideableEntity) SetPositionMovement(prevX, prevY, prevZ, nextX, nextY, nextZ float64, yaw byte) {
	r.MovementState.PrevX = prevX
	r.MovementState.PrevY = prevY
	r.MovementState.PrevZ = prevZ
	r.MovementState.X = nextX
	r.MovementState.Y = nextY
	r.MovementState.Z = nextZ
	r.Yaw = yaw
	r.MovementState.PositionChanged = true
}

func (r *RideableEntity) Despawn() bool {
	if r.ShouldDespawn {
		r.ShouldDespawn = false
		return true
	}
	return false
}

func (r *RideableEntity) IsItem() bool { return false }

func (r *RideableEntity) GetVelocity() (float64, float64, float64) {
	return r.VelocityX, r.VelocityY, r.VelocityZ
}

func (r *RideableEntity) IsMob() bool {
	return false
}

func (r *RideableEntity) GetDim() int32 {
	return r.Dimension
}

func (r *RideableEntity) GetLoggedIn() bool {
	return true
}

func (r *RideableEntity) SetPosition(x, y, z float64) {
	r.X, r.Y, r.Z = x, y, z
}

func (r *RideableEntity) SetHP(hp int16) {
	r.HP = hp
}

func (r *RideableEntity) GetHP() int16 {
	return r.HP
}

func (r *RideableEntity) IsPlayer() bool {
	return false
}

func (r *RideableEntity) GetEntityId() int32 {
	return r.EntityId
}

func (r *RideableEntity) GetPosition() (float64, float64, float64) {
	return r.X, r.Y, r.Z
}

func (r *RideableEntity) IsRideable() bool {
	return true
}

func (r *RideableEntity) GetName() string {
	return fmt.Sprintf("Entity %d", r.EntityId)
}

type RidableAction int

const (
	Moved RidableAction = iota
	Stopped
	Despawned
)

func (e *RideableEntity) Tick(
	getBlock GetBlockFunc,
	players []PlayerPosition,
) (newX, newY, newZ float64, yaw byte, action RidableAction) {
	if e.HP <= 0 {
		return e.X, e.Y, e.Z, e.Yaw, Despawned
	}

	e.tickCollision(players)

	switch e.ObjectType {
	case constants.ObjectBoat:
		return e.TickBoat(getBlock)
	case constants.ObjectMinecart:
		return e.TickMinecart(getBlock)
	}
	return 0, 0, 0, 0, Despawned
}

func (e *RideableEntity) tickCollision(players []PlayerPosition) {
	e.applyRiderInput()
	const pushRadius, pushForce = 1.25, 0.3
	for _, pp := range players {
		if pp.EntityId == e.PassengerEntityId {
			continue
		}
		dx := e.X - pp.X
		dy := e.Y - pp.Y
		dz := e.Z - pp.Z
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
		if dist < pushRadius && dist > 0.001 {
			nx, nz := dx/dist, dz/dist
			if nx*e.VelocityX+nz*e.VelocityZ >= 0 {
				e.VelocityX += nx * pushForce
				e.VelocityZ += nz * pushForce
			}
		}
	}
}
