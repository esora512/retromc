package entities

import (
	"math"

	"github.com/leNicDev/retromc/constants"
)

type BlockEntity struct {
	X             int32
	Y             float64
	Z             int32
	TypeId        int16
	Metadata      byte
	EntityId      int32
	VelocityY     float64
	Landed        bool
	IsFalling     bool
	Dimension     int32
	ShouldDespawn bool
	MovementState constants.MovementState
	ObjectType    byte
}

func (b *BlockEntity) GetEntityType() constants.EntityType {
	return constants.FallingBlock
}

func (b *BlockEntity) GetMovementState() *constants.MovementState {
	return &b.MovementState
}

func (b *BlockEntity) SetVelocityMovement(vx, vy, vz float64) {
	b.MovementState.VelocityX = vx
	b.MovementState.VelocityY = vy
	b.MovementState.VelocityZ = vz
	//b.MovementState.VelocityChanged = true
}

func (r *BlockEntity) SetPositionMovement(prevX, prevY, prevZ, nextX, nextY, nextZ float64, yaw byte) {
	r.MovementState.PrevX = prevX
	r.MovementState.PrevY = prevY
	r.MovementState.PrevZ = prevZ
	r.MovementState.X = nextX
	r.MovementState.Y = nextY
	r.MovementState.Z = nextZ
	r.MovementState.PositionAndRotationChanged = true
}

func (r *BlockEntity) SetTeleportMovement(nextX, nextY, nextZ float64, yaw byte) {
	r.MovementState.X = nextX
	r.MovementState.Y = nextY
	r.MovementState.Z = nextZ
	r.MovementState.Teleported = true
}

func NewFallingBlockEntity(eId int32, x int32, y float64, z int32, typeId int16, metadata byte, dim int32, oType byte) *BlockEntity {
	e := &BlockEntity{
		EntityId: eId,
		X:        x, Y: y, Z: z,
		TypeId: typeId, Metadata: metadata,
		Dimension:     dim,
		VelocityY:     0,
		ObjectType:    oType,
		IsFalling:     false,
		ShouldDespawn: false,
	}
	e.SetVelocityMovement(0, e.VelocityY, 0)
	return e
}

func (b *BlockEntity) Despawn() bool {
	if b.ShouldDespawn {
		b.ShouldDespawn = false
		return true
	}
	return false
}

func (b *BlockEntity) GetVelocity() (float64, float64, float64) {
	return 0, b.VelocityY, 0
}

func (b *BlockEntity) GetDim() int32 {
	return b.Dimension
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

func (b *BlockEntity) GetEntityId() int32 {
	return b.EntityId
}

func (b *BlockEntity) SetHP(hp int16) {
	// No-op
}

func (b *BlockEntity) GetHP() int16 {
	return -1
}

func (e *BlockEntity) Tick(getBlock func(x int32, y byte, z int32) constants.WBlock) {
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
		if beneath.IsSolid() {
			e.Landed = true
			e.Y = float64(groundY)
			e.VelocityY = 0
			return
		}
	}
	e.Y = newY
}
