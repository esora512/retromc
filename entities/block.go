package entities

import (
	"math"
)

type BlockEntity struct {
	X            int32
	Y            float64
	Z            int32
	TypeId       int16
	Metadata     byte
	EntityId     int32
	VelocityY    float64
	Landed       bool
	VelocitySent bool
}

func NewBlockEntity(entityId int32, typeId int16, metadata byte, x, y, z float64) *BlockEntity {
	return &BlockEntity{
		EntityId:     entityId,
		TypeId:       typeId,
		Metadata:     metadata,
		X:            int32(x),
		Y:            float64(y),
		Z:            int32(z),
		VelocityY:    0,
		VelocitySent: false,
	}
}

func (b *BlockEntity) GetLoggedIn() bool {
	return true
}

func (b *BlockEntity) GetName() string {
	return "BlockEntity"
}

func (b *BlockEntity) GetPosition() (float64, float64, float64) {
	return float64(b.X), float64(b.Y), float64(b.Z)
}

func (b *BlockEntity) SetPosition(x, y, z float64) {
	b.X = int32(x)
	b.Y = float64(y)
	b.Z = int32(z)
}

func (b *BlockEntity) IsRideable() bool {
	return false
}

func (b *BlockEntity) GetEntityId() int32 {
	return b.EntityId
}

func (b *BlockEntity) IsPlayer() bool {
	return false
}

func (b *BlockEntity) SetHP(hp int16) {
	// No-op
}

func (b *BlockEntity) GetHP() int16 {
	return 0
}

func (e *BlockEntity) TickBlock(getBlock func(x int32, y byte, z int32) BlockInfo) {
	if e.Landed {
		return
	}

	e.VelocityY -= 0.04
	e.VelocityY *= 0.98

	newY := e.Y + e.VelocityY

	if newY < 0 {
		e.Landed = true
		e.Y = 0
		return
	}

	groundY := int32(math.Floor(newY))
	if groundY >= 1 {
		beneath := getBlock(int32(e.X), byte(groundY-1), int32(e.Z))
		if beneath.IsSolid {
			e.Landed = true
			e.Y = float64(groundY)
			e.VelocityY = 0
			return
		}
	}
	e.Y = newY
}
