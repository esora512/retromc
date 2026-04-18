package entities

import "fmt"

type RideableEntity struct {
	EntityId      int32
	X             float64
	Y             float64
	Z             float64
	VelocityX     float64
	VelocityY     float64
	VelocityZ     float64
	OwnerEntityId int32
	ObjectType    byte
}

func (r *RideableEntity) SetPosition(x, y, z float64) {
	r.X, r.Y, r.Z = x, y, z
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
