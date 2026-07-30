package main

import (
	"bufio"
	"flag"
	"log"
	"math"
	"net"
	"os"
	"time"

	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/packethandler"
	"github.com/leNicDev/retromc/player"
)

const (
	CON_TYPE = "tcp"
)

var (
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func main() {
	host := flag.String("host", "localhost", "Address to bind the server to")
	port := flag.String("port", "25565", "Port to bind the server to")
	flag.Parse()
	l, err := net.Listen(CON_TYPE, *host+":"+*port)
	if err != nil {
		log.Panicln("Failed to bind to address", err.Error())
	}

	// close listener when the application closes
	defer l.Close()

	log.Printf("Server listening on %s:%s (PID: %d)", *host, *port, os.Getpid())

	world := level.NewWorld(GitCommit, 0, level.Default)
	entityTracker := level.NewEntityTracker(packets.SpawnPlayerEntityPacket, packets.SpawnObjectPacket, packets.EntityDespawnPacket, packets.SetEquipment2)
	gameLoop(world, entityTracker)
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	for {
		// listen for incoming connections
		connection, err := l.Accept()
		if err != nil {
			log.Fatalln("Failed to accept connection: ", err.Error())
			continue
		}
		// handle connection
		go handleConnection(connection, world, entityTracker)
	}

}

func handleKeepAlive(connection net.Conn, stop chan struct{}) {
	// send keep-alive every 10s so the client doesn't time out
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		keepAlive := packets.KeepAliveOutPacket{}
		for {
			select {
			case <-ticker.C:
				_, err := connection.Write(keepAlive.Serialize())
				if err != nil {
					return
				}
			case <-stop:
				return
			}
		}
	}()
}

func handleConnection(connection net.Conn, world *level.World, tracker *level.EntityTracker) {
	pl := player.NewPlayer(connection)
	done := make(chan struct{})
	handleKeepAlive(connection, done)

	reader := bufio.NewReader(connection)
	for {
		err := packethandler.HandlePacket(connection, reader, world, pl, tracker)
		if err != nil {
			//log.Println("Connection closed:", err.Error())
			log.Println("Connection closed...")

			if pl.Username != "" {
				unlock := world.LockSession(pl.Username)
				defer unlock()
				if cur, ok := world.GetPlayerByUsername(pl.Username); !ok || cur == pl {
					pData := level.ToPlayerData(pl)
					if saveErr := level.SavePlayerData(world.WorldDir, pl.Username, pData); saveErr != nil {
						log.Println("Failed to save inventory:", saveErr)
					}
					if saveErr := level.SaveMcRegion(world, world.WorldDir); saveErr != nil {
						log.Println("Failed to save the world:", saveErr)
					}
					world.BroadcastPacket(packets.PlayerEntityDespawnPacket(pl))
					world.RemovePlayer(pl)
					tracker.Remove(pl.GetEntityId())
					world.UnloadPlayerChunks(pl)
				}
			}
			close(done)
			connection.Close()
			return
		}
	}
}

func gameLoop(world *level.World, entityTracker *level.EntityTracker) {
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			// For fast time, set it to TickSpeed to 20
			world.Tick = (world.Tick + world.TickSpeed) % 24000
			world.BroadcastTime()
			world.TickFluids(world.Tick)
			world.TickFallables(world.Tick)
			fallingBlocksPhysics(world)
			ridablePhysics(world)
			world.GrowPhysics()
			packethandler.CollectNearbyItems(world)
			packethandler.ApplyGravityOnDroppedItems(world)
			furnaceLogic(world)
			world.UnloadUnusedChunks()
			if world.Tick%10 == 0 {
				entityTracker.Manage(world)
			}
			world.FlushBlockQueue()
		}
	}()
}

func fallingBlocksPhysics(world *level.World) {
	toRemove := []int32{}
	allEntities := world.SnapshotEntities()

	for _, e := range allEntities {
		falling, ok := e.(*entities.BlockEntity)
		if !ok {
			continue
		}
		if !world.IsLoaded(falling.X, falling.Z) {
			continue
		}

		if !falling.IsFalling {
			if areaLoaded(world, falling.X, falling.Z, 32) {
				falling.IsFalling = true
			} else {
				instaFall(world, falling)
				toRemove = append(toRemove, falling.EntityId)
				continue
			}
		}

		if !falling.VelocitySent {
			packethandler.BroadcastEntityVelocity(world, falling.EntityId, 0, falling.VelocityY, 0)
			falling.VelocitySent = true
		}

		falling.TickBlock(func(x int32, y byte, z int32) entities.BlockInfo {
			b := world.GetBlock(x, y, z)
			return entities.BlockInfo{
				IsSolid:  !b.IsAir() && !b.IsLiquid() && !b.IsSnowLayer(),
				Metadata: int(b.Metadata),
			}
		})

		if falling.Landed && falling.Y >= 0 {
			toRemove = append(toRemove, falling.EntityId)
			block := level.NewBlockById(falling.TypeId, falling.Metadata)
			world.SetBlockInQueue(falling.X, int32(falling.Y), falling.Z, block)
			//packethandler.SetBlockAndNotify(world, falling.X, int32(falling.Y), falling.Z, &block)
		}
	}

	for _, id := range toRemove {
		world.RemoveEntity(id)
		despawn := packets.EntityDespawnOutPacket{EntityId: id}
		world.BroadcastPacket(despawn.Serialize())
	}
}

func makeSendFurnaceProgress(world *level.World) func(progress, fuelMax, fuelRemain int) {
	return func(progress, fuelDuration, fuelRemain int) {
		p1 := packets.ContainerDataOutPacket{
			WindowID: 1,
			Type:     0,
			Value:    int16(progress),
		}
		p2 := packets.ContainerDataOutPacket{
			WindowID: 1,
			Type:     1,
			Value:    int16(fuelRemain),
		}
		p3 := packets.ContainerDataOutPacket{
			WindowID: 1,
			Type:     2,
			Value:    int16(fuelDuration),
		}
		world.BroadcastPacket(p1.Serialize())
		world.BroadcastPacket(p2.Serialize())
		world.BroadcastPacket(p3.Serialize())
	}
}

func makeSendFurnaceSlot(world *level.World) func(item inventory.Item, slot int16) {
	return func(item inventory.Item, slot int16) {
		p := packets.SetSlotOutPacket{
			WindowId: 1,
			Slot:     slot,
			Item:     item,
		}
		world.BroadcastPacket(p.Serialize())
	}
}

func makeSetFurnaceBlock(world *level.World) func(x, y, z int16, lit bool) {
	return func(x, y, z int16, lit bool) {
		oldBlock := world.GetBlock(int32(x), byte(y), int32(z))

		var newBlock level.Block
		if lit {
			newBlock = level.NewLitFurnaceBlock(oldBlock.Metadata)
		} else {
			newBlock = level.NewFurnaceBlock(oldBlock.Metadata)
		}
		world.SetBlockInQueue(int32(x), int32(y), int32(z), newBlock)
		//packethandler.SetBlockAndNotify(world, int32(x), int32(y), int32(z), &newBlock)
	}
}

func furnaceLogic(world *level.World) {
	furnaces := world.GetAllFurnaces()
	inventory.TickFurnaces(furnaces, makeSendFurnaceProgress(world), makeSendFurnaceSlot(world), makeSetFurnaceBlock(world))
}

func ridablePhysics(world *level.World) {
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

	getBlock := func(x int32, y byte, z int32) entities.BlockInfo {
		b := world.GetBlock(x, y, z)
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
		nx, ny, nz, yaw, action := ridable.TickPhysics(getBlock, players)

		switch action {
		case entities.Moved:
			packethandler.BroadcastRelativePosition(world, ridable, cx, cy, cz, nx, ny, nz, yaw)
			ridable.SetPosition(nx, ny, nz)

			velX := nx - cx
			velY := ny - cy
			velZ := nz - cz
			maybeBroadcastVelocity(world, ridable, velX, velY, velZ)

		case entities.Stopped:
			packethandler.BroadcastTeleport(world, ridable, cx, cy, cz, yaw)
			maybeBroadcastVelocity(world, ridable, 0, 0, 0)

		case entities.Despawned:
			despawn := packets.EntityDespawnOutPacket{EntityId: ridable.EntityId}
			world.BroadcastPacket(despawn.Serialize())
			toRemove = append(toRemove, ridable.EntityId)
		}
	}

	for _, id := range toRemove {
		world.RemoveEntity(id)
	}
}

func maybeBroadcastVelocity(world *level.World, ridable *entities.RideableEntity, vx, vy, vz float64) {
	const epsilon = 0.02

	dx := vx - ridable.LastSentVelX
	dy := vy - ridable.LastSentVelY
	dz := vz - ridable.LastSentVelZ

	if math.Abs(dx) < epsilon && math.Abs(dy) < epsilon && math.Abs(dz) < epsilon {
		return
	}

	packethandler.BroadcastEntityVelocity(world, ridable.EntityId, vx, vy, vz)

	ridable.LastSentVelX = vx
	ridable.LastSentVelY = vy
	ridable.LastSentVelZ = vz
	ridable.VelocityX, ridable.VelocityY, ridable.VelocityZ = vx, vy, vz
}

func areaLoaded(world *level.World, x, z, radius int32) bool {
	offsets := []int32{-radius, 0, radius}
	for _, dx := range offsets {
		for _, dz := range offsets {
			if !world.IsLoaded(x+dx, z+dz) {
				return false
			}
		}
	}
	return true
}

func instaFall(world *level.World, falling *entities.BlockEntity) {
	x, z := falling.X, falling.Z
	y := int32(falling.Y)

	for y > 0 {
		below := world.GetBlock(x, byte(y-1), z)
		if below.IsSnowLayer() {
			y--
			break
		}
		if !below.IsAir() && !below.IsLiquid() {
			break
		}
		y--
	}

	block := level.NewBlockById(falling.TypeId, falling.Metadata)
	world.SetBlockInQueue(x, y, z, block)
}
