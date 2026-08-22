package entities

import (
	"math"
)

const (
	RiderInputAcceleration = 0.5
	RiderTurnVelocityBlend = 0.5
	YawSmoothing           = 0.5
	BoatYOffset            = 0.6
	BoatYOffsetWater       = 0.3
)

func (boat *RideableEntity) applyRiderInput() {
	vx := boat.PassengerVelocityX
	vz := boat.PassengerVelocityZ

	boat.VelocityX += vx * RiderInputAcceleration
	boat.VelocityZ += vz * RiderInputAcceleration

	riderSpeedSq := vx*vx + vz*vz
	if riderSpeedSq <= 1e-4 {
		return
	}

	speed := math.Sqrt(boat.VelocityX*boat.VelocityX + boat.VelocityZ*boat.VelocityZ)
	if speed <= 0.01 {
		return
	}

	riderSpeed := math.Sqrt(riderSpeedSq)
	targetVX := vx / riderSpeed * speed
	targetVZ := vz / riderSpeed * speed

	boat.VelocityX += (targetVX - boat.VelocityX) * RiderTurnVelocityBlend
	boat.VelocityZ += (targetVZ - boat.VelocityZ) * RiderTurnVelocityBlend

	desiredYaw := math.Atan2(-targetVZ, -targetVX) * 180.0 / math.Pi
	boat.YawDegrees += wrapDegrees(desiredYaw-boat.YawDegrees) * YawSmoothing
}

func wrapDegrees(angle float64) float64 {
	for angle >= 180.0 {
		angle -= 360.0
	}
	for angle < -180.0 {
		angle += 360.0
	}
	return angle
}

func (boat *RideableEntity) TickBoat(
	w WorldShared,
) (newX, newY, newZ float64, yaw byte, action RidableAction) {
	const (
		maxSpeed      = 0.2
		maxSpeedWater = 0.3
		gravity       = 0.04
		waterFriction = 0.97
		landFriction  = 0.85
	)

	// Water detection
	bx := int32(math.Floor(boat.X))
	bz := int32(math.Floor(boat.Z))
	feetBlockY := int32(math.Floor(boat.Y - BoatYOffset))
	onWater := false
	if feetBlockY >= 0 && feetBlockY < 128 {
		feetBlock := w.GetBlock(bx, byte(feetBlockY), bz, boat.Dimension)
		onWater = feetBlock.IsWater()
	}

	speed := maxSpeed
	if onWater {
		speed = maxSpeedWater
	}
	boat.VelocityX = clamp(boat.VelocityX, -speed, speed)
	boat.VelocityZ = clamp(boat.VelocityZ, -speed, speed)

	// Gravity / buoyancy
	if onWater {
		targetY := float64(feetBlockY+1) + BoatYOffsetWater
		boat.VelocityY = (targetY - boat.Y) * 0.2
	} else {
		boat.VelocityY -= gravity
	}

	newX = boat.X + boat.VelocityX
	newY = boat.Y + boat.VelocityY
	newZ = boat.Z + boat.VelocityZ

	// Ground collision
	bx = int32(math.Floor(newX))
	bz = int32(math.Floor(newZ))
	feetY := newY - BoatYOffset
	groundY := int32(math.Floor(feetY - 0.001))
	if groundY >= 0 && groundY < 128 {
		below := w.GetBlock(bx, byte(groundY), bz, boat.Dimension)
		if below.IsSolid() {
			newY = float64(groundY) + 1 + BoatYOffset
			boat.VelocityY = 0
		}
	}

	// Horizontal collision
	bodyBlockY := int32(math.Floor(newY - BoatYOffset))
	if bodyBlockY >= 0 && bodyBlockY < 128 {
		ox := int32(math.Floor(boat.X))
		oz := int32(math.Floor(boat.Z))
		fx := int32(math.Floor(newX))
		b := w.GetBlock(fx, byte(bodyBlockY), oz, boat.Dimension)
		if fx != ox && b.IsSolid() {
			newX = boat.X
			boat.VelocityX = 0
		}
		fz := int32(math.Floor(newZ))
		nx := int32(math.Floor(newX))
		b = w.GetBlock(nx, byte(bodyBlockY), fz, boat.Dimension)
		if fz != oz && b.IsSolid() {
			newZ = boat.Z
			boat.VelocityZ = 0
		}
	}

	// Friction
	friction := landFriction
	if onWater {
		friction = waterFriction
	}
	boat.VelocityX *= friction
	boat.VelocityZ *= friction

	// Dead-zone snap
	if math.Abs(boat.VelocityX) < 0.001 {
		boat.VelocityX = 0
	}
	if math.Abs(boat.VelocityZ) < 0.001 {
		boat.VelocityZ = 0
	}

	dx := boat.X - newX
	dz := boat.Z - newZ
	if dx*dx+dz*dz > 0.001 {
		desiredYaw := math.Atan2(dz, dx) * 180.0 / math.Pi
		boat.YawDegrees += wrapDegrees(desiredYaw-boat.YawDegrees) * YawSmoothing
	}
	yaw = byte(int(math.Round(boat.YawDegrees*256.0/360.0)) & 0xFF)
	boat.Yaw = yaw

	return newX, newY, newZ, yaw, Moved

}
