package entities

import "math"

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

type BlockInfo struct {
	IsRail        bool
	IsPoweredRail bool
	Metadata      int
}

type GetBlockFunc func(x int32, y byte, z int32) BlockInfo

type CartAction int

const (
	CartMoved CartAction = iota
	CartStopped
	CartDespawned
)

type PlayerPosition struct{ X, Y, Z float64 }

func (cart *RideableEntity) TickPhysics(
	getBlock GetBlockFunc,
	players []PlayerPosition,
) (newX, newY, newZ float64, yaw byte, action CartAction) {
	const maxSpeed = 0.4

	cx, cy, cz := cart.GetPosition()

	bx := int32(math.Floor(cx))
	by := int32(math.Floor(cy))
	bz := int32(math.Floor(cz))

	// check one below, like the original
	block := getBlock(bx, byte(by-1), bz)
	if block.IsRail {
		by--
	}

	block = getBlock(bx, byte(by), bz)
	if !block.IsRail {
		return 0, 0, 0, 0, CartDespawned
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
	if block.IsPoweredRail {
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

	dirs := railDirs[meta]
	dirX := float64(dirs[1][0] - dirs[0][0])
	dirZ := float64(dirs[1][2] - dirs[0][2])
	dirLen := math.Sqrt(dirX*dirX + dirZ*dirZ)

	p1x := float64(bx) + 0.5 + float64(dirs[0][0])*0.5
	p1z := float64(bz) + 0.5 + float64(dirs[0][2])*0.5
	p2x := float64(bx) + 0.5 + float64(dirs[1][0])*0.5
	p2z := float64(bz) + 0.5 + float64(dirs[1][2])*0.5
	segDX := p2x - p1x
	segDZ := p2z - p1z

	// snap cart position onto the rail segment, matching Java's onUpdate snap
	var t float64
	if segDX == 0.0 {
		cx = float64(bx) + 0.5
		t = cz - float64(bz)
	} else if segDZ == 0.0 {
		cz = float64(bz) + 0.5
		t = cx - float64(bx)
	} else {
		t = ((cx-p1x)*segDX + (cz-p1z)*segDZ) * 2.0
	}
	cx = p1x + segDX*t
	cz = p1z + segDZ*t

	// now align velocity
	dot := cart.VelocityX*dirX + cart.VelocityZ*dirZ
	if dot < 0 {
		dirX, dirZ = -dirX, -dirZ
	}
	speed := math.Sqrt(cart.VelocityX*cart.VelocityX + cart.VelocityZ*cart.VelocityZ)
	cart.VelocityX = speed * dirX / dirLen
	cart.VelocityZ = speed * dirZ / dirLen

	// then move
	nextX := cx + cart.VelocityX
	nextZ := cz + cart.VelocityZ

	// player push
	const pushRadius, pushForce = 1.25, 0.3
	for _, pp := range players {
		dx := cx - pp.X
		dy := cy - pp.Y
		dz := cz - pp.Z
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
		if dist < pushRadius && dist > 0.001 {
			nx, nz := dx/dist, dz/dist
			if nx*cart.VelocityX+nz*cart.VelocityZ >= 0 {
				cart.VelocityX += nx * pushForce
				cart.VelocityZ += nz * pushForce
			}
		}
	}

	// align velocity to rail direction using the table
	dirs = railDirs[meta]
	dirX = float64(dirs[1][0] - dirs[0][0])
	dirZ = float64(dirs[1][2] - dirs[0][2])
	dirLen = math.Sqrt(dirX*dirX + dirZ*dirZ)

	dot = cart.VelocityX*dirX + cart.VelocityZ*dirZ
	if dot < 0 {
		dirX, dirZ = -dirX, -dirZ
	}

	speed = math.Sqrt(cart.VelocityX*cart.VelocityX + cart.VelocityZ*cart.VelocityZ)
	cart.VelocityX = speed * dirX / dirLen
	cart.VelocityZ = speed * dirZ / dirLen

	// powered rail: boost or brake
	if block.IsPoweredRail {
		isActivated := true
		//isActivated := (block.Metadata & 8) != 0
		if isActivated {
			// boost
			if speed > 0.01 {
				cart.VelocityX += cart.VelocityX / speed * 0.06
				cart.VelocityZ += cart.VelocityZ / speed * 0.06
			}
		} else {
			// brake — unpowered powered rail
			if speed < 0.03 {
				cart.VelocityX = 0
				cart.VelocityY = 0
				cart.VelocityZ = 0
			} else {
				cart.VelocityX *= 0.5
				cart.VelocityY = 0
				cart.VelocityZ *= 0.5
			}
		}
	}

	// cap speed
	cart.VelocityX = clamp(cart.VelocityX, -maxSpeed, maxSpeed)
	cart.VelocityZ = clamp(cart.VelocityZ, -maxSpeed, maxSpeed)

	// move
	nextX = cx + cart.VelocityX
	nextZ = cz + cart.VelocityZ

	// get Y before and after for hill momentum transfer
	_, prevY, _, hasPrev := getRailPos(getBlock, cx, cy, cz)
	rx, nextY, rz, hasNext := getRailPos(getBlock, nextX, cy, nextZ)

	if hasNext {
		nextX = rx
		nextZ = rz
		// hill momentum: going downhill adds speed, uphill removes it
		if hasPrev {
			slope := (prevY - nextY) * 0.05
			speed = math.Sqrt(cart.VelocityX*cart.VelocityX + cart.VelocityZ*cart.VelocityZ)
			if speed > 0 {
				cart.VelocityX = cart.VelocityX / speed * (speed + slope)
				cart.VelocityZ = cart.VelocityZ / speed * (speed + slope)
			}
		}
	} else {
		// stop cart when it is about to go off-rails
		cart.VelocityX = 0
		cart.VelocityZ = 0
		cart.VelocityY = 0
		return cx, cy, cz, 0, CartStopped
	}

	// friction: 0.96 unoccupied (0.997 if rider — add that check if you have passenger support)
	cart.VelocityX *= 0.96
	cart.VelocityZ *= 0.96
	cart.VelocityY = 0 // Y motion zeroed while on rail

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

	return nextX, nextY, nextZ, yaw, CartMoved
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

func getRailPos(bInfo GetBlockFunc, px, py, pz float64) (float64, float64, float64, bool) {
	bx := int32(math.Floor(px))
	by := int32(math.Floor(py))
	bz := int32(math.Floor(pz))

	block := bInfo(bx, byte(by), bz)
	if !block.IsRail {
		block = bInfo(bx, byte(by-1), bz)
		if !block.IsRail {
			return 0, 0, 0, false
		}
		by--
	}

	meta := int(block.Metadata)
	if block.IsPoweredRail {
		meta &= 7
	}

	dirs := railDirs[meta]
	// endpoints of the rail segment in world space
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
		px = float64(bx) + 0.5
		t = pz - float64(bz)
	} else if dz == 0.0 {
		pz = float64(bz) + 0.5
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
