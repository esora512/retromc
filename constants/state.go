package constants

type MovementState struct {
	X, Y, Z             float64
	PrevX, PrevY, PrevZ float64

	VelocityX, VelocityY, VelocityZ float64

	PositionAndRotationChanged bool
	PositionChanged bool 
	RotationChanged bool
	VelocityChanged            bool
	Teleported                 bool

	Yaw   byte
	Pitch byte
}
