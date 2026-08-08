package level

import (
	"math"

	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/inventory"
)

const (
	pickupRangeSq = 1.5 * 1.5
)

func (w *World) DroppedItemPhysics() {
	w.CollectNearbyItems()
	w.GravityOnItems()
}

func (w *World) GravityOnItems() {
	chunks := w.PlayerActiveChunks(1, 0)
	for _, chunk := range chunks {
		logic := chunk.Logic
		for _, dropped := range logic.DroppedItems {
			below := w.GetBlock(int32(dropped.X), byte(dropped.Y)-1, int32(dropped.Z), 0)
			if below.IsAir() || below.IsLiquid() {
				dropped.Y--
			}
		}
	}

	nchunks := w.PlayerActiveChunks(1, -1)
	for _, chunk := range nchunks {
		logic := chunk.Logic
		for _, dropped := range logic.DroppedItems {
			below := w.GetBlock(int32(dropped.X), byte(dropped.Y)-1, int32(dropped.Z), -1)
			if below.IsAir() || below.IsLiquid() {
				dropped.Y--
			}
		}
	}

}

func (world *World) CollectNearbyItems() {
	chunks := world.PlayerActiveChunks(1, 0) // 3x3 chunks around each player
	chunks = append(chunks, world.PlayerActiveChunks(1, -1)...)
	for _, chunk := range chunks {
		logic := chunk.Logic
		for entityId, dropped := range logic.DroppedItems {
			if dropped.PickupDelay > 0 {
				dropped.PickupDelay--
				continue
			}
			itemX := float64(dropped.X)
			itemY := float64(dropped.Y)
			itemZ := float64(dropped.Z)

			for _, pl := range world.Players {
				dx := pl.X - itemX
				dy := pl.Y - itemY
				dz := pl.Z - itemZ
				if dx*dx+dy*dy+dz*dz > pickupRangeSq {
					continue
				}

				slot := pl.Inventory.AddItem(int16(dropped.ItemId), uint16(dropped.Metadata), dropped.Amount)
				if slot < 0 {
					continue
				}
				t := pl.Inventory.Items[slot]
				world.SendSetSlot(pl.Connection, 0, slot, t)

				collect := world.CollectItem(entityId, int32(pl.GetEntityId()))
				world.BroadcastPacket(collect)
				world.RemoveDroppedItem(entityId, dropped.X, dropped.Z)
				break
			}
		}
	}
}

func (world *World) FallingBlocksPhysics() {
	toRemove := []int32{}
	allEntities := world.SnapshotEntities()

	for _, e := range allEntities {
		falling, ok := e.(*entities.BlockEntity)
		if !ok {
			continue
		}
		if !world.IsLoaded(falling.X, falling.Z, falling.Dimension) {
			continue
		}

		if !falling.IsFalling {
			if world.areaLoaded(falling.X, falling.Z, 32, falling.Dimension) {
				falling.IsFalling = true
			} else {
				world.instaFall(falling, falling.Dimension)
				toRemove = append(toRemove, falling.EntityId)
				continue
			}
		}

		if !falling.VelocitySent {
			world.BroadcastEntityVelocity(falling.EntityId, 0, falling.VelocityY, 0)
			falling.VelocitySent = true
		}

		falling.Tick(func(x int32, y byte, z int32) entities.BlockInfo {
			b := world.GetBlock(x, y, z, falling.Dimension)
			return entities.BlockInfo{
				IsSolid:  !b.IsAir() && !b.IsLiquid() && !b.IsSnowLayer(),
				Metadata: int(b.Metadata),
			}
		})

		if falling.Landed && falling.Y >= 0 {
			toRemove = append(toRemove, falling.EntityId)
			block := NewBlockById(falling.TypeId, falling.Metadata)
			world.SetBlockInQueue(falling.X, int32(falling.Y), falling.Z, block, falling.Dimension)
		}
	}

	for _, id := range toRemove {
		world.RemoveEntity(id)
		world.BroadcastDespawn(id)
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

	block := NewBlockById(typeId, metadata)
	world.SetBlockInQueue(x, y, z, block, dim)
}

func (world *World) instaFall(falling *entities.BlockEntity, dim int32) {
	world.instaFallAt(falling.X, falling.Z, int32(falling.Y), falling.TypeId, falling.Metadata, dim)
}

func (world *World) maybeBroadcastVelocity(ridable *entities.RideableEntity, vx, vy, vz float64) {
	const epsilon = 0.02

	dx := vx - ridable.LastSentVelX
	dy := vy - ridable.LastSentVelY
	dz := vz - ridable.LastSentVelZ

	if math.Abs(dx) < epsilon && math.Abs(dy) < epsilon && math.Abs(dz) < epsilon {
		return
	}

	world.BroadcastEntityVelocity(ridable.EntityId, vx, vy, vz)

	ridable.LastSentVelX = vx
	ridable.LastSentVelY = vy
	ridable.LastSentVelZ = vz
	ridable.VelocityX, ridable.VelocityY, ridable.VelocityZ = vx, vy, vz
}

func (world *World) RidablePhysics(tacker *EntityTracker) {
	allEntities := world.SnapshotEntities()
	var ridables []*entities.RideableEntity
	var players []entities.PlayerPosition

	for _, e := range allEntities {
		if e.IsPlayer() {
			x, y, z := e.GetPosition()
			players = append(players, entities.PlayerPosition{X: x, Y: y, Z: z, EntityId: e.GetEntityId()})
		} else if ridable, ok := e.(*entities.RideableEntity); ok {
			if ridable.ObjectType == 1 || ridable.ObjectType == 10 {
				ridables = append(ridables, ridable)
			}
		}
	}

	getBlock := func(x int32, y byte, z int32, dim int32) entities.BlockInfo {
		b := world.GetBlock(x, y, z, dim)
		return entities.BlockInfo{
			IsRail:        b.IsRail(),
			IsPoweredRail: b.IsPoweredRail(),
			IsSolid:       !b.IsAir() && !b.IsLiquid(),
			Metadata:      int(b.Metadata),
			IsWater:       b.IsWater(),
		}
	}

	var toRemove []int32

	for _, ridable := range ridables {
		cx, cy, cz := ridable.GetPosition()
		nx, ny, nz, yaw, action := ridable.Tick(getBlock, players)

		switch action {
		case entities.Moved:
			world.BroadcastRelativePosition(ridable, cx, cy, cz, nx, ny, nz, yaw)
			ridable.SetPosition(nx, ny, nz)

			velX := nx - cx
			velY := ny - cy
			velZ := nz - cz
			world.maybeBroadcastVelocity(ridable, velX, velY, velZ)

		case entities.Stopped:
			world.BroadcastTeleport(ridable, cx, cy, cz, yaw)
			world.maybeBroadcastVelocity(ridable, 0, 0, 0)

		case entities.Despawned:
			world.BroadcastDespawn(ridable.EntityId)
			toRemove = append(toRemove, ridable.EntityId)
		}
	}

	for _, id := range toRemove {
		world.RemoveEntity(id)
		tacker.Remove(id)
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

		var newBlock Block
		if lit {
			newBlock = NewLitFurnaceBlock(oldBlock.Metadata)
		} else {
			newBlock = NewFurnaceBlock(oldBlock.Metadata)
		}
		w.SetBlockInQueue(int32(x), int32(y), int32(z), newBlock, dim)
	}
}

func (w *World) TickFurnaces() {
	furnaces := w.GetAllFurnaces()
	inventory.TickFurnaces(furnaces, w.makeSendFurnaceProgress(), w.makeSendFurnaceSlot(), w.makeSetFurnaceBlock())
}

func (w *World) AdvanceTick(nextTick int64, tracker *EntityTracker) {
	w.Tick = nextTick
	w.AdvanceTime()
	w.TickFluids()
	w.TickFallables()
	w.TickLeaves()
	w.FallingBlocksPhysics()
	w.RidablePhysics(tracker)
	w.GrowPhysics()
	w.DroppedItemPhysics()
	w.TickFurnaces()
	w.TickSleep()
	w.TickPlayers()
	w.TickMobs()
}

func (w *World) TickSleep() {
	w.Sleep()
	w.SleepThroughNight()
}

func (w *World) TickPlayers() {
	for _, pl := range w.Players {
		pl.OnlineFor += 1
	}
}

func (w *World) TickMobs() {
	for _, e := range w.Entities {
		if m, ok := e.(*Mob); ok {
			m.Move(w)
		}
	}
}