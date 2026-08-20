package constants

type MovementState struct {
	X, Y, Z             float64
	PrevX, PrevY, PrevZ float64

	VelocityX, VelocityY, VelocityZ float64

	PositionAndRotationChanged bool
	PositionChanged            bool
	RotationChanged            bool
	VelocityChanged            bool
	Teleported                 bool

	Yaw   float32
	Pitch float32
}

type EntityType int

const (
	Player EntityType = iota
	Mob
	Ridable
	DroppedItem
	FallingBlock
)

type Entity interface {
	GetName() string
	GetPosition() (float64, float64, float64)
	SetPosition(x, y, z float64)
	GetEntityId() int32
	SetHP(hp int16)
	GetHP() int16
	GetLoggedIn() bool
	GetDim() int32
	GetVelocity() (float64, float64, float64)
	Despawn() bool
	GetMovementState() *MovementState
	GetEntityType() EntityType
}
