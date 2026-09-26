package level

import (
	"math"
	"math/rand"

	"github.com/leNicDev/retromc/constants"
	c "github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/player"
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
	w.ItemPhysicsTick()
	w.CollectNearbyItems()
}

func (w *World) CollectNearbyItems() {
	for _, e := range w.Entities {
		d, ok := e.(*entities.DroppedItem)
		if !ok {
			continue
		}

		// the pickup delay is counted down by the item's own tick
		if d.PickupDelay > 0 || d.Dead || d.CollectorId != -1 {
			continue
		}

		x, y, z := d.GetPosition()
		dim := d.GetDim()

		for _, pl := range w.Players {
			if pl.HP <= 0 {
				continue
			}
			if dim != pl.Dimension {
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

			it := inventory.NewItem(int16(d.ItemId), d.Amount, d.Metadata)
			touched := pl.Inventory.PickupItem(&it)
			for _, slot := range touched {
				w.SendSetSlot(pl.Connection, 0, slot, pl.Inventory.Items[slot])
				if slot == pl.HotbarSlot {
					for _, v := range w.Players {
						if v != pl && v.LoggedIn {
							w.SetEquipment(pl, v)
						}
					}
				}
			}
			if it.Count > 0 {
				d.Amount = it.Count
				continue
			}
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

		if falling.Landed {
			falling.ShouldDespawn = true
			block := constants.NewBlockById(falling.TypeId, falling.Metadata)
			world.SetBlockInQueue(falling.X, int32(falling.Y), falling.Z, block, falling.Dimension)
		}

		if falling.Y < 0 {
			falling.ShouldDespawn = true
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

	for _, ridable := range ridables {
		cx, cy, cz := ridable.GetPosition()
		nx, ny, nz, yaw, action := ridable.Tick(world, players)
		ridable.MovementState.KeepRotation = true

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

func (w *World) forEachFurnaceViewer(furnace *inventory.Furnace, fn func(pl *player.Player)) {
	w.ForEachPlayer(func(pl *player.Player) {
		if pl.InventoryType != player.FurnaceInventory {
			return
		}
		if w.GetFurnace(pl.Furnace.X, pl.Furnace.Y, pl.Furnace.Z, pl.Furnace.Dim) == furnace {
			fn(pl)
		}
	})
}

func (w *World) makeSendFurnaceProgress() func(furnace *inventory.Furnace, progress, fuelMax, fuelRemain int) {
	return func(furnace *inventory.Furnace, progress, fuelDuration, fuelRemain int) {
		w.forEachFurnaceViewer(furnace, func(pl *player.Player) {
			w.SendContainerData(pl.Connection, 1, 0, int16(progress))
			w.SendContainerData(pl.Connection, 1, 1, int16(fuelRemain))
			w.SendContainerData(pl.Connection, 1, 2, int16(fuelDuration))
		})
	}
}

func (w *World) makeSendFurnaceSlot() func(furnace *inventory.Furnace, item inventory.Item, slot int16) {
	return func(furnace *inventory.Furnace, item inventory.Item, slot int16) {
		w.forEachFurnaceViewer(furnace, func(pl *player.Player) {
			w.SendSetSlot(pl.Connection, 1, slot, item)
		})
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
	w.RandomTickPhysics()
	w.DroppedItemPhysics()
	w.TickFurnaces()
	w.TickSleep()
	w.TickPlayers()
	w.TickMobs(tracker)
	w.SpawnHostiles()
	w.SpawnAnimals()
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
	var mobs []*entities.Mob
	for _, e := range w.Entities {
		if m, ok := e.(*entities.Mob); ok {
			mobs = append(mobs, m)
		}
	}

	remove := func(m *entities.Mob) {
		tracker.SendToViewers(w, m.EntityId, w.DespawnEntity(m.EntityId))
		w.RemoveEntity(m.EntityId)
		tracker.ResetEntity(m.EntityId)
	}

	for _, m := range mobs {
		if _, ok := w.Entities[m.EntityId]; !ok {
			continue
		}
		if m.GetHP() <= 0 {
			if m.DespawnIn < 0 {
				m.DespawnIn = 21
			}
			continue
		}

		closestSq := math.MaxFloat64
		for _, p := range w.Players {
			if !p.LoggedIn || p.GetDim() != m.Dimension {
				continue
			}
			dx, dy, dz := m.X-p.X, m.Y-p.Y, m.Z-p.Z
			closestSq = math.Min(closestSq, dx*dx+dy*dy+dz*dz)
		}
		if closestSq == math.MaxFloat64 {
			continue
		}

		if closestSq > 128*128 {
			remove(m)
			continue
		}
		if m.Age > 600 && rand.Intn(800) == 0 {
			if closestSq < 32*32 {
				m.Age = 0
			} else {
				remove(m)
				continue
			}
		}

		m.Move(w, tracker)
	}
}

func (w *World) SendHealth(entityId int32, newHp int16) {
	pl, ok := w.Players[entityId]
	if !ok {
		return
	}
	w.sendSetHealth(pl.Connection, uint16(newHp))
}

func (w *World) SpawnHostiles() {
	if !w.IsNight() {
		return
	}
	w.spawnMobsAroundPlayers(func(x, y, z, dim int32) {
		w.SpawnMobType(randomType(hostileTypes), x, y, z, dim, -1)
	})
}

func (w *World) SpawnAnimals() {
	if w.IsNight() {
		return
	}
	w.spawnMobsAroundPlayers(func(x, y, z, dim int32) {
		w.SpawnMobType(randomType(animalTypes), x, y, z, dim, -1)
	})
}

func (w *World) spawnMobsAroundPlayers(spawn func(x, y, z, dim int32)) {
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
		sx, sz := int32(math.Floor(spawnX)), int32(math.Floor(spawnZ))
		spawnY, ok := w.findGroundY(sx, sz, int32(py), dim)
		if !ok {
			continue
		}

		spawn(sx, spawnY, sz, dim)
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
	if !w.IsLoaded(x, z, dim) {
		return 0, false
	}
	free := func(y int32) bool {
		b := w.GetBlock(x, byte(y), z, dim)
		return !b.IsSolid() && !b.IsLiquid()
	}
	for y := min(startY+searchRange/2, WorldMaxY-2); y > startY-searchRange && y > 1; y-- {
		below := w.GetBlock(x, byte(y-1), z, dim)
		if entities.IsNormalCube(below) && free(y) && free(y+1) {
			return y, true
		}
	}
	return 0, false
}
