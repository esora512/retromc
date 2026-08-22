package level

import (
	"math"
	"math/rand"

	"github.com/leNicDev/retromc/constants"
	c "github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/inventory"
)

const (
	pickupRangeSq = 1.5 * 1.5
	pickupRangeY  = 2.5
)

func (w *World) ItemPhysicsTick() {
	for _, e := range w.Entities {
		d, ok := e.(*entities.DroppedItem)
		if !ok {
			continue
		}
		d.Tick(w)
	}
}

func (w *World) DroppedItemPhysics() {
	w.CollectNearbyItems()
	w.ItemPhysicsTick()
}

func (w *World) CollectNearbyItems() {
	for _, e := range w.Entities {
		d, ok := e.(*entities.DroppedItem)
		if !ok {
			continue
		}

		if d.PickupDelay > 0 {
			d.PickupDelay--
			continue
		}

		x, y, z := d.GetPosition()

		for _, pl := range w.Players {
			if pl.HP <= 0 {
				continue
			}

			dx := pl.X - x
			dz := pl.Z - z
			dy := pl.Y - y

			if dx*dx+dz*dz > pickupRangeSq {
				continue
			}
			if dy < 0 {
				dy = -dy
			}
			if dy > pickupRangeY {
				continue
			}

			slot := pl.Inventory.AddItem(int16(d.ItemId), uint16(d.Metadata), d.Amount)
			if slot < 0 {
				continue
			}
			t := pl.Inventory.Items[slot]
			w.SendSetSlot(pl.Connection, 0, slot, t)
			d.CollectorId = pl.GetEntityId()
			break
		}
	}
}

var fallingBlockSafeRadius = int32(VIEW_DISTANCE * 16 / 2)

func (world *World) FallingBlocksPhysics() {
	allEntities := world.SnapshotEntities()

	for _, e := range allEntities {
		falling, ok := e.(*entities.BlockEntity)
		if !ok {
			continue
		}
		if !world.IsLoaded(falling.X, falling.Z, falling.Dimension) {
			continue
		}

		falling.IsFalling = true

		falling.Tick(func(x int32, y byte, z int32) constants.WBlock {
			return world.GetBlock(x, y, z, falling.Dimension)
		})

		if falling.Landed && falling.Y >= 0 {
			falling.ShouldDespawn = true
			block := constants.NewBlockById(falling.TypeId, falling.Metadata)
			world.SetBlockInQueue(falling.X, int32(falling.Y), falling.Z, block, falling.Dimension)
		}
	}
}

func (world *World) areaLoaded(x, z, radius int32, dim int32) bool {
	offsets := []int32{-radius, 0, radius}
	for _, dx := range offsets {
		for _, dz := range offsets {
			if !world.IsLoaded(x+dx, z+dz, dim) {
				return false
			}
		}
	}
	return true
}

func (world *World) instaFallAt(x, z, startY int32, typeId int16, metadata byte, dim int32) {
	y := startY
	for y > 0 {
		below := world.GetBlock(x, byte(y-1), z, dim)
		if below.IsSnowLayer() {
			y--
			break
		}
		if !below.IsAir() && !below.IsLiquid() {
			break
		}
		y--
	}

	block := constants.NewBlockById(typeId, metadata)
	world.SetBlockInQueue(x, y, z, block, dim)
}

func maybeSetVelocityMovement(ridable *entities.RideableEntity, vx, vy, vz float64) {
	const epsilon = 0.02

	dx := vx - ridable.LastSentVelX
	dy := vy - ridable.LastSentVelY
	dz := vz - ridable.LastSentVelZ

	if math.Abs(dx) < epsilon && math.Abs(dy) < epsilon && math.Abs(dz) < epsilon {
		return
	}

	ridable.SetVelocityMovement(vx, vy, vz)

	ridable.LastSentVelX = vx
	ridable.LastSentVelY = vy
	ridable.LastSentVelZ = vz
	ridable.VelocityX, ridable.VelocityY, ridable.VelocityZ = vx, vy, vz
}

func (world *World) RidablePhysics() {
	allEntities := world.SnapshotEntities()
	var ridables []*entities.RideableEntity
	var players []entities.PlayerPosition

	for _, e := range allEntities {
		switch e.GetEntityType() {
		case c.Player:
			x, y, z := e.GetPosition()
			players = append(players, entities.PlayerPosition{X: x, Y: y, Z: z, EntityId: e.GetEntityId()})
		case c.Ridable:
			ridable, _ := e.(*entities.RideableEntity)
			if ridable.ObjectType == 1 || ridable.ObjectType == 10 {
				ridables = append(ridables, ridable)
			}
		default:
			continue
		}
	}

	getBlock := func(x int32, y byte, z int32, dim int32) constants.WBlock {
		b := world.GetBlock(x, y, z, dim)
		return b
	}
	for _, ridable := range ridables {
		cx, cy, cz := ridable.GetPosition()
		nx, ny, nz, yaw, action := ridable.Tick(getBlock, players)

		switch action {
		case entities.Moved:
			ridable.SetPositionMovement(cx, cy, cz, nx, ny, nz, yaw)
			ridable.SetPosition(nx, ny, nz)
			ridable.MovementState.Pitch = 0

			velX := nx - cx
			velY := ny - cy
			velZ := nz - cz
			maybeSetVelocityMovement(ridable, velX, velY, velZ)

		case entities.Stopped:
			ridable.SetTeleportMovement(cx, cy, cz, yaw)
			maybeSetVelocityMovement(ridable, 0, 0, 0)
		}
	}
}

func (w *World) makeSendFurnaceProgress() func(progress, fuelMax, fuelRemain int) {
	return func(progress, fuelDuration, fuelRemain int) {
		w.BroadcastContainerData(1, 0, int16(progress))
		w.BroadcastContainerData(1, 1, int16(fuelRemain))
		w.BroadcastContainerData(1, 2, int16(fuelDuration))
	}
}

func (w *World) makeSendFurnaceSlot() func(item inventory.Item, slot int16) {
	return func(item inventory.Item, slot int16) {
		w.BroadcastSetSlot(1, slot, item)
	}
}

func (w *World) makeSetFurnaceBlock() func(x, y, z int16, lit bool, dim int32) {
	return func(x, y, z int16, lit bool, dim int32) {
		oldBlock := w.GetBlock(int32(x), byte(y), int32(z), dim)

		var newBlock constants.WBlock
		if lit {
			newBlock = constants.NewLitFurnaceBlock(oldBlock.Metadata)
		} else {
			newBlock = constants.NewFurnaceBlock(oldBlock.Metadata)
		}
		w.SetBlockInQueue(int32(x), int32(y), int32(z), newBlock, dim)
	}
}

func (w *World) TickFurnaces() {
	furnaces := w.GetAllFurnaces()
	inventory.TickFurnaces(furnaces, w.makeSendFurnaceProgress(), w.makeSendFurnaceSlot(), w.makeSetFurnaceBlock())
}

func (w *World) AdvanceTick(nextTick int64, tracker *entities.EntityTracker) {
	w.Tick = nextTick
	w.AdvanceTime()
	w.TickFluids()
	w.TickFallables()
	w.TickLeaves()
	w.FallingBlocksPhysics()
	w.RidablePhysics()
	w.GrowPhysics()
	w.DroppedItemPhysics()
	w.TickFurnaces()
	w.TickSleep()
	w.TickPlayers()
	w.TickMobs(tracker)
	w.SpawnSpiders()
}

func (w *World) TickSleep() {
	w.Sleep()
	w.SleepThroughNight()
}

func (w *World) TickPlayers() {
	for _, pl := range w.Players {
		if pl.Immune >= 0 {
			pl.Immune--
		}
	}
}

func (w *World) TickMobs(tracker *entities.EntityTracker) {
	for _, e := range w.Entities {
		if m, ok := e.(*entities.Mob); ok {
			m.Move(w, tracker)
		}
	}
}

func (w *World) SendHealth(entityId int32, newHp int16) {
	pl, ok := w.Players[entityId]
	if !ok {
		return
	}
	w.sendSetHealth(pl.Connection, uint16(newHp))
}

func (w *World) SpawnSpiders() {
	if !w.IsNight() {
		return
	}
	count := 0
	for _, e := range w.Entities {
		if _, ok := e.(*entities.Mob); ok {
			count++
		}
	}

	if count >= 16 {
		return
	}

	for _, pl := range w.Players {
		if count >= 16 {
			break
		}

		px, py, pz := pl.GetPosition()
		dim := pl.GetDim()

		spawnX, spawnZ := randomPointOnRing(px, pz, 48)
		spawnY, ok := w.findGroundY(int32(spawnX), int32(spawnZ), int32(py), dim)
		if !ok {
			return
		}

		w.SpawnSpider(int32(spawnX), spawnY, int32(spawnZ), dim, -1)
		count++
	}
}

func randomPointOnRing(px, pz float64, dist float64) (x, z float64) {
	angle := rand.Float64() * 2 * math.Pi
	baseX := px + math.Cos(angle)*dist
	baseZ := pz + math.Sin(angle)*dist

	jitterX := rand.Float64()*16 - 8
	jitterZ := rand.Float64()*16 - 8

	return baseX + jitterX, baseZ + jitterZ
}

func (w *World) findGroundY(x, z, startY, dim int32) (int32, bool) {
	const searchRange = 32
	for y := startY; y > startY-searchRange && y > 0; y-- {
		b := w.GetBlock(x, byte(y), z, dim)
		if b.IsSolid() {
			return y + 1, true
		} else {
			return 0, false
		}
	}
	return startY, true
}
