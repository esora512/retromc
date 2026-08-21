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

	UntrackVelocityIn            int
	UntrackPositionIn            int
	UntrackPositionAndRotationIn int
	UntrackRotationIn            int

	IsHurt   bool
	ArmSwing bool
}

func (m *MovementState) VChanged() bool {
	if m.UntrackVelocityIn > 0 {
		m.UntrackVelocityIn--
		return true
	}
	return m.VelocityChanged
}

func (m *MovementState) PosChanged() bool {
	if m.UntrackPositionIn > 0 {
		m.UntrackPositionIn--
		return true
	}
	return m.PositionChanged
}

func (m *MovementState) RotChanged() bool {
	if m.UntrackRotationIn > 0 {
		m.UntrackRotationIn--
		return true
	}
	return m.RotationChanged
}

func (m *MovementState) PosAndRotChanged() bool {
	if m.UntrackPositionAndRotationIn > 0 {
		m.UntrackPositionAndRotationIn--
		return true
	}
	return m.PositionAndRotationChanged
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
