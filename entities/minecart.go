package entities

import (
	"math"
)

var railDirs = [10][2][3]int{
	0: {{0, 0, -1}, {0, 0, 1}},
	1: {{-1, 0, 0}, {1, 0, 0}},
	2: {{-1, -1, 0}, {1, 0, 0}},
	3: {{-1, 0, 0}, {1, -1, 0}},
	4: {{0, 0, -1}, {0, -1, 1}},
	5: {{0, -1, -1}, {0, 0, 1}},
	6: {{0, 0, 1}, {1, 0, 0}},
	7: {{0, 0, 1}, {-1, 0, 0}},
	8: {{0, 0, -1}, {-1, 0, 0}},
	9: {{0, 0, -1}, {1, 0, 0}},
}

type PlayerPosition struct {
	X, Y, Z  float64
	EntityId int32
}

func (cart *RideableEntity) TickMinecart(
	getBlock GetBlock,
) (newX, newY, newZ float64, yaw byte, action RidableAction) {
	const maxSpeed = 0.4

	cx, cy, cz := cart.GetPosition()

	bx := int32(math.Floor(cx))
	by := int32(math.Floor(cy))
	bz := int32(math.Floor(cz))

	// check one below, like the original
	block := getBlock(bx, byte(by-1), bz, cart.Dimension)
	if block.IsRail() {
		by--
	}

	block = getBlock(bx, byte(by), bz, cart.Dimension)
	if !block.IsRail() {
		return 0, 0, 0, 0, Despawned
	}

	// Notchian off-rail behaviour
	// NOTE: Not implementing it because do not wanna deal with off-rail minecarts
	// TODO: Impelement it maybe in future
	// if !block.IsRail {
	// 	cart.VelocityY -= 0.04
	// 	cart.VelocityX = clamp(cart.VelocityX, -maxSpeed, maxSpeed)
	// 	cart.VelocityZ = clamp(cart.VelocityZ, -maxSpeed, maxSpeed)
	// 	cx += cart.VelocityX
	// 	cy += cart.VelocityY
	// 	cz += cart.VelocityZ
	// 	cart.VelocityX *= 0.95
	// 	cart.VelocityY *= 0.95
	// 	cart.VelocityZ *= 0.95
	// 	return cx, cy, cz, CartMoved
	// }

	// strip powered-rail activation bit to get shape meta
	meta := block.Metadata
	if block.IsPoweredRail() {
		meta &= 7
	}

	// slope gravity nudge — exact values from original
	switch meta {
	case 2:
		cart.VelocityX -= 1.0 / 128.0
	case 3:
		cart.VelocityX += 1.0 / 128.0
	case 4:
		cart.VelocityZ += 1.0 / 128.0
	case 5:
		cart.VelocityZ -= 1.0 / 128.0
	}

	// align velocity
	dirs := railDirs[meta]
	dirX := float64(dirs[1][0] - dirs[0][0])
	dirZ := float64(dirs[1][2] - dirs[0][2])
	dirLen := math.Sqrt(dirX*dirX + dirZ*dirZ)
	dot := cart.VelocityX*dirX + cart.VelocityZ*dirZ
	if dot < 0.0 {
		dirX = -dirX
		dirZ = -dirZ
	}
	speed := math.Sqrt(cart.VelocityX*cart.VelocityX + cart.VelocityZ*cart.VelocityZ)
	cart.VelocityX = speed * dirX / dirLen
	cart.VelocityZ = speed * dirZ / dirLen

	railStartX := float64(bx) + 0.5 + float64(dirs[0][0])*0.5
	railStartZ := float64(bz) + 0.5 + float64(dirs[0][2])*0.5
	railEndX := float64(bx) + 0.5 + float64(dirs[1][0])*0.5
	railEndZ := float64(bz) + 0.5 + float64(dirs[1][2])*0.5
	railDirX := railEndX - railStartX
	railDirZ := railEndZ - railStartZ

	// Java's opUpdate, line 265 - 289
	posAlongRail := 0.0
	if railDirX == 0.0 {
		posAlongRail = cz - float64(bz)
	} else if railDirZ == 0.0 {
		posAlongRail = cx - float64(bx)
	} else {
		posAlongRail = ((cx-railStartX)*railDirX + (cz-railStartZ)*railDirZ) * 2.0
		// NOTE: Potential fix, reduce velocity when hitting curves; but does the server do that?
	}

	cx = railStartX + railDirX*posAlongRail
	cz = railStartZ + railDirZ*posAlongRail

	var nextX, nextZ float64

	// Powered rail boost / braking
	if block.IsPoweredRail() {
		isActivated := true
		// TODO: When redstone is implemented, uncomment line below
		//isActivated := (block.Metadata & 8) != 0
		if isActivated {
			if speed > 0.01 {
				cart.VelocityX += cart.VelocityX / speed * 0.06
				cart.VelocityZ += cart.VelocityZ / speed * 0.06

				cart.MovementState.VelocityX = cart.VelocityX
				cart.MovementState.VelocityZ = cart.VelocityZ
			}
		} else {
			// brake — unpowered powered rail
			if speed < 0.03 {
				cart.VelocityX = 0
				cart.VelocityY = 0
				cart.VelocityZ = 0

				cart.MovementState.VelocityX = cart.VelocityX
				cart.MovementState.VelocityZ = cart.VelocityZ
			} else {
				cart.VelocityX *= 0.5
				cart.VelocityY = 0
				cart.VelocityZ *= 0.5

				cart.MovementState.VelocityX = cart.VelocityX
				cart.MovementState.VelocityZ = cart.VelocityZ
			}
		}
	}

	cart.VelocityX = clamp(cart.VelocityX, -maxSpeed, maxSpeed)
	cart.VelocityZ = clamp(cart.VelocityZ, -maxSpeed, maxSpeed)

	cart.MovementState.VelocityX = cart.VelocityX
	cart.MovementState.VelocityZ = cart.VelocityZ

	nextX = cx + cart.VelocityX
	nextZ = cz + cart.VelocityZ

	// get Y before and after for hill momentum transfer
	_, prevY, _, hasPrev := getRailPos(getBlock, cx, cy, cz, cart.Dimension)
	_, nextY, _, hasNext := getRailPos(getBlock, nextX, cy, nextZ, cart.Dimension)

	if hasNext && hasPrev {
		slope := (prevY - nextY) * 0.05
		speed = math.Sqrt(cart.VelocityX*cart.VelocityX + cart.VelocityZ*cart.VelocityZ)
		if speed > 0 {
			cart.VelocityX = cart.VelocityX / speed * (speed + slope)
			cart.VelocityZ = cart.VelocityZ / speed * (speed + slope)

			cart.MovementState.VelocityX = cart.VelocityX
			cart.MovementState.VelocityZ = cart.VelocityZ
		}
	} else {
		// stop cart when it is about to go off-rails
		cart.VelocityX = 0
		cart.VelocityZ = 0
		cart.VelocityY = 0

		cart.MovementState.VelocityX = cart.VelocityX
		cart.MovementState.VelocityZ = cart.VelocityZ
		return cx, cy, cz, 0, Stopped
	}

	// friction: 0.96 unoccupied
	cart.VelocityX *= 0.96
	cart.VelocityZ *= 0.96
	cart.VelocityY = 0 // Y motion zeroed while on rail

	cart.MovementState.VelocityX = cart.VelocityX
	cart.MovementState.VelocityZ = cart.VelocityZ

	if math.Abs(cart.VelocityX) < 0.001 {
		cart.VelocityX = 0
	}
	if math.Abs(cart.VelocityZ) < 0.001 {
		cart.VelocityZ = 0
	}

	dx := cx - nextX
	dz := cz - nextZ
	if dx*dx+dz*dz > 0.001 {
		degrees := math.Atan2(dz, dx) * 180.0 / math.Pi
		yaw = byte(int(degrees*256.0/360.0) & 0xFF)
	}
	return nextX, nextY, nextZ, yaw, Moved
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func getRailPos(bInfo GetBlock, px, py, pz float64, dim int32) (float64, float64, float64, bool) {
	bx := int32(math.Floor(px))
	by := int32(math.Floor(py))
	bz := int32(math.Floor(pz))

	block := bInfo(bx, byte(by-1), bz, dim)
	if block.IsRail() {
		by--
	} else {
		block = bInfo(bx, byte(by), bz, dim)
		if !block.IsRail() {
			return 0, 0, 0, false
		}
	}

	meta := int(block.Metadata)
	if block.IsPoweredRail() {
		meta &= 7
	}

	dirs := railDirs[meta]
	x1 := float64(bx) + 0.5 + float64(dirs[0][0])*0.5
	y1 := float64(by) + 0.5 + float64(dirs[0][1])*0.5
	z1 := float64(bz) + 0.5 + float64(dirs[0][2])*0.5
	x2 := float64(bx) + 0.5 + float64(dirs[1][0])*0.5
	y2 := float64(by) + 0.5 + float64(dirs[1][1])*0.5
	z2 := float64(bz) + 0.5 + float64(dirs[1][2])*0.5

	dx := x2 - x1
	dy := (y2 - y1) * 2.0 // doubled like the original
	dz := z2 - z1

	var t float64
	if dx == 0.0 {
		t = pz - float64(bz)
	} else if dz == 0.0 {
		t = px - float64(bx)
	} else {
		t = ((px-x1)*dx + (pz-z1)*dz) * 2.0
	}

	rx := x1 + dx*t
	ry := y1 + dy*t
	rz := z1 + dz*t

	// original adjusts Y based on dy direction
	if dy < 0.0 {
		ry += 1.0
	}
	if dy > 0.0 {
		ry += 0.5
	}

	return rx, ry, rz, true
}
