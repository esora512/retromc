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


