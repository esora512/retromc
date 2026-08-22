package entities

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/leNicDev/retromc/constants"
)

type DroppedItem struct {
	EntityId    int32
	ItemId      int32
	Amount      byte
	Metadata    byte
	X, Y, Z     float64
	PickupDelay int32
	Dim         int32

	VelX, VelY, VelZ float64

	DespawnIn     int
	MovementState constants.MovementState
	InLava        bool

	CollectorId int32
}

func (d *DroppedItem) GetEntityType() constants.EntityType {
	return constants.DroppedItem
}

func (d *DroppedItem) GetMovementState() *constants.MovementState {
	return &d.MovementState
}

func (d *DroppedItem) Despawn() bool {
	if d.DespawnIn < 0 {
		return false
	}
	if d.DespawnIn == 0 {
		d.DespawnIn = -1
		return true
	}
	d.DespawnIn -= 1
	return false
}

func (d *DroppedItem) GetEntityId() int32 {
	return d.EntityId
}

func (d *DroppedItem) GetHP() int16 {
	return 20
}

func (d *DroppedItem) SetHP(hp int16) {}

func (d *DroppedItem) GetName() string {
	return fmt.Sprintf("Entity %d", d.EntityId)
}

func (d *DroppedItem) GetPosition() (float64, float64, float64) {
	return float64(d.X), float64(d.Y), float64(d.Z)
}

func (d *DroppedItem) SetPosition(x, y, z float64) {}

func (d *DroppedItem) GetLoggedIn() bool { return false }

func (d *DroppedItem) GetDim() int32 { return d.Dim }

func (d *DroppedItem) GetVelocity() (float64, float64, float64) { return d.VelX, d.VelY, d.VelZ }

// Vibed dropped item fluid handling because can't be bothered with math and it works, lel
const itemColliderHalfWidth = 0.125
const itemColliderHeight = 0.25

func itemAABB(d *DroppedItem) (minX, minY, minZ, maxX, maxY, maxZ float64) {
	return d.X - itemColliderHalfWidth, d.Y, d.Z - itemColliderHalfWidth,
		d.X + itemColliderHalfWidth, d.Y + itemColliderHeight, d.Z + itemColliderHalfWidth
}

func fluidHeight(b constants.WBlock) float64 {
	if b.IsStillWater() || b.IsStillLava() {
		return 1.0
	}
	level := int(b.Metadata)
	if level > 7 {
		level = 7
	}
	return float64(7-level) / 8.0
}

type FlowVec struct{ X, Z float64 }

var lateralOffsets = []struct{ dx, dz int32 }{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

const (
	gravity         = 0.04
	airDrag         = 0.98
	groundDragBase  = 0.58800006
	lavaBounceY     = 0.2
	lavaJitterScale = 0.2
	bounceFactor    = -0.5

	waterFlowStrength = 0.014
	lavaFlowStrength  = 0.1
	buoyancy          = 0.02
	buoyancyMaxUp     = 0.06
	lavaLifetime      = 3
)


func getFlowVector(w WorldShared, bx int32, by byte, bz int32, dim int32, b constants.WBlock) FlowVec {
	isLava := b.IsLava()
	ownHeight := fluidHeight(b)

	var flow FlowVec
	for _, n := range lateralOffsets {
		nx, nz := bx+n.dx, bz+n.dz
		if !w.IsLoaded(nx, nz, dim) {
			continue
		}
		nb := w.GetBlock(nx, by, nz, dim)

		var nHeight float64
		switch {
		case isLava && nb.IsLava():
			nHeight = fluidHeight(nb)
		case !isLava && nb.IsWater():
			nHeight = fluidHeight(nb)
		case nb.IsFluidReplaceable():
			// air/flowers etc: if fluid continues one block down, treat this
			// direction as a steep drop so flow pulls toward the edge/fall
			if by == 0 {
				continue
			}
			below := w.GetBlock(nx, by-1, nz, dim)
			if (isLava && below.IsLava()) || (!isLava && below.IsWater()) {
				nHeight = ownHeight - 1.0
			} else {
				continue
			}
		default:
			continue
		}

		diff := ownHeight - nHeight
		flow.X += float64(n.dx) * diff
		flow.Z += float64(n.dz) * diff
	}

	if length := math.Hypot(flow.X, flow.Z); length > 1e-4 {
		flow.X /= length
		flow.Z /= length
	}
	return flow
}

func handleFluidAcceleration(d *DroppedItem, w WorldShared, wantLava bool) bool {
	minX, minY, minZ, maxX, maxY, maxZ := itemAABB(d)

	bx0, bx1 := int32(math.Floor(minX)), int32(math.Floor(maxX))
	by0, by1 := int32(math.Floor(minY)), int32(math.Floor(maxY))
	bz0, bz1 := int32(math.Floor(minZ)), int32(math.Floor(maxZ))

	touched := false
	var accumX, accumZ float64
	var accumY float64

	for bx := bx0; bx <= bx1; bx++ {
		for by := by0; by <= by1; by++ {
			if by < 0 || by > 255 {
				continue
			}
			for bz := bz0; bz <= bz1; bz++ {
				if !w.IsLoaded(bx, bz, d.Dim) {
					continue
				}
				b := w.GetBlock(bx, byte(by), bz, d.Dim)
				isLava := b.IsLava()
				isWater := b.IsWater()
				if (wantLava && !isLava) || (!wantLava && !isWater) {
					continue
				}

				h := fluidHeight(b)
				blockTop := float64(by) + h
				if blockTop < minY {
					continue
				}
				touched = true

				flow := getFlowVector(w, bx, byte(by), bz, d.Dim, b)
				accumX += flow.X
				accumZ += flow.Z
				accumY += 1.0
			}
		}
	}

	if !touched {
		return false
	}

	if length := math.Hypot(accumX, accumZ); length > 1e-4 {
		accumX /= length
		accumZ /= length
	}

	strength := waterFlowStrength
	if wantLava {
		strength = lavaFlowStrength
	}

	d.VelX += accumX * strength
	d.VelZ += accumZ * strength
	if accumY > 0 && d.VelY < buoyancyMaxUp {
		d.VelY += buoyancy
	}
	return true
}

func (d *DroppedItem) Tick(w WorldShared) {
	if d.PickupDelay > 0 {
		d.PickupDelay--
	}

	inWater := handleFluidAcceleration(d, w, false)
	inLava := handleFluidAcceleration(d, w, true)

	if inLava {
		if d.DespawnIn < 0 {
			d.DespawnIn = lavaLifetime
		}
		d.VelY = lavaBounceY
		d.VelX = float64(rand.Float32()-rand.Float32()) * lavaJitterScale
		d.VelZ = float64(rand.Float32()-rand.Float32()) * lavaJitterScale
	} else {
		d.VelY -= gravity
	}
	_ = inWater
	d.X += d.VelX
	d.Y += d.VelY
	d.Z += d.VelZ

	blockAtFeet := w.GetBlock(
		int32(math.Floor(d.X)), byte(math.Floor(d.Y)), int32(math.Floor(d.Z)), d.Dim,
	)

	onGround := false
	if !blockAtFeet.IsAir() && !blockAtFeet.IsLiquid() && d.VelY <= 0 {
		onGround = true
		d.Y = math.Floor(d.Y) + 1
	}

	drag := float64(airDrag)
	if onGround {
		drag = groundDragBase
		d.VelY *= bounceFactor
	}

	d.VelX *= drag
	d.VelZ *= drag
	d.VelY *= 0.9800000190734863
}
