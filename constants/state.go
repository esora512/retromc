package constants

type MovementState struct {
	X, Y, Z             float64
	PrevX, PrevY, PrevZ float64

	VelocityX, VelocityY, VelocityZ float64

	PositionChanged bool
	VelocityChanged bool
	Teleported bool

	Yaw byte
	Pitch byte
}
