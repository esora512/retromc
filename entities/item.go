package entities

import (
	"fmt"

	"github.com/leNicDev/retromc/constants"
)

type DroppedItem struct {
	EntityId    int32
	ItemId      int32
	Amount      byte
	Metadata    byte
	X, Y, Z     float64
	PickupDelay int32
	Dim         int32

	VelX, VelY, VelZ float64

	DespawnIn     int
	MovementState constants.MovementState
	InLava bool

	CollectorId int32
}

func (d *DroppedItem) GetEntityType() constants.EntityType {
	return constants.DroppedItem
}

func (d *DroppedItem) GetMovementState() *constants.MovementState {
	return &d.MovementState
}

func (d *DroppedItem) Despawn() bool {
	if d.DespawnIn < 0 {
		return false
	}
	if d.DespawnIn == 0 {
		d.DespawnIn = -1
		return true
	}
	d.DespawnIn -= 1
	return false
}

func (d *DroppedItem) GetEntityId() int32 {
	return d.EntityId
}

func (d *DroppedItem) GetHP() int16 {
	return 20
}

func (d *DroppedItem) SetHP(hp int16) {}

func (d *DroppedItem) GetName() string {
	return fmt.Sprintf("Entity %d", d.EntityId)
}

func (d *DroppedItem) GetPosition() (float64, float64, float64) {
	return float64(d.X), float64(d.Y), float64(d.Z)
}

func (d *DroppedItem) SetPosition(x, y, z float64) {}

func (d *DroppedItem) GetLoggedIn() bool { return false }

func (d *DroppedItem) GetDim() int32 { return d.Dim }

func (d *DroppedItem) GetVelocity() (float64, float64, float64) { return d.VelX, d.VelY, d.VelZ }
