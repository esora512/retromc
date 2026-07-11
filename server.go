package main

import (
	"bufio"
	"log"
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
	CON_HOST = "localhost"
	CON_PORT = "25565"
	CON_TYPE = "tcp"
)

func main() {
	l, err := net.Listen(CON_TYPE, CON_HOST+":"+CON_PORT)
	if err != nil {
		log.Panicln("Failed to bind to address", err.Error())
	}

	// close listener when the application closes
	defer l.Close()

	log.Printf("Server listening on %s:%s (PID: %d)", CON_HOST, CON_PORT, os.Getpid())

	world := level.NewWorld(level.Template)
	if err := world.LoadChanges(worldSavePath); err != nil {
		log.Println("Failed to load world save:", err)
	}
	if err := world.LoadContainers(containerSavePath); err != nil {
		log.Println("Failed to load container save:", err)
	}
	gameLoop(world)
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
		go handleConnection(connection, world)
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

func handleConnection(connection net.Conn, world *level.World) {
	pl := player.NewPlayer(connection)
	done := make(chan struct{})
	handleKeepAlive(connection, done)

	world.AddPlayer(pl)
	reader := bufio.NewReader(connection)
	for {
		err := packethandler.HandlePacket(connection, reader, world, pl)
		//log.Printf("Player Health for %s: %d", pl.Username, pl.Health)
		if err != nil {
			log.Println("Connection closed:", err.Error())
			if pl.Username != "" {
				if saveErr := player.SaveInventory(pl.Username, pl.Inventory); saveErr != nil {
					log.Println("Failed to save inventory:", saveErr)
				}
			}
			world.BroadcastPacket(packets.PlayerEntityDespawnPacket(pl))
			world.RemovePlayer(pl)
			close(done)
			connection.Close()
			return
		}
	}
}

const worldSavePath = "saves/world.dat"
const containerSavePath = "saves/containers.dat"

func gameLoop(world *level.World) {
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			// For fast time, set it to TickSpeed to 20
			world.Tick = (world.Tick + world.TickSpeed) % 24000
			world.BroadcastTime()
			fallingBlocksPhysics(world)
			ridablePhysics(world)
			world.CleanUpFallable()
			world.GrowPhysics()
			packethandler.CollectNearbyItems(world)
			packethandler.ApplyGravityOnDroppedItems(world)
			level.CheckLavaHarden(world, packethandler.SetBlockAndNotify)
			if world.Tick%20 == 0 || world.Tick%60 == 0 {
				waterCfg := level.NewWaterConfig(world)
				lavaCfg := level.NewLavaConfig(world)
				level.FluidDecay(world, waterCfg, packethandler.SetBlockAndNotify)
				level.FluidSpreading(world, waterCfg, packethandler.SetBlockAndNotify)
				level.InfiniteWaterSource(world, waterCfg, packethandler.SetBlockAndNotify)
				level.RefreshSourceBlocks(world, waterCfg, packethandler.SetBlockAndNotify)
				level.FluidDecay(world, lavaCfg, packethandler.SetBlockAndNotify)
				if world.Tick%60 == 0 {
					level.FluidSpreading(world, lavaCfg, packethandler.SetBlockAndNotify)
				}
			}

			furnaceLogic(world)
			world.UnloadUnusedChunks()

			// Save world every 1200 ticks = every 60s
			// if world.Tick%1200 == 0 {
			// 	if err := world.SaveChanges(worldSavePath); err != nil {
			// 		log.Println("Failed to save world:", err)
			// 	}
			// 	if err := inventory.SaveContainers(containerSavePath); err != nil {
			// 		log.Println("Failed to save containers:", err)
			// 	}
			// }
		}
	}()
}

func fallingBlocksPhysics(world *level.World) {
	toRemove := []int32{}
	allEntities := world.SnapshotEntities()
	for _, e := range allEntities {
		if falling, ok := e.(*entities.BlockEntity); ok {
			prevX := float64(falling.X)
			prevY := falling.Y
			prevZ := float64(falling.Z)

			falling.TickBlock(func(x int32, y byte, z int32) entities.BlockInfo {
				b := world.GetBlock(x, y, z)
				return entities.BlockInfo{
					IsSolid:  !b.IsAir() && !b.IsLiquid(),
					Metadata: int(b.Metadata),
				}
			})

			if falling.Landed {
				if falling.Y >= 0 {
					toRemove = append(toRemove, falling.EntityId)
					block := level.NewBlockById(falling.TypeId, falling.Metadata)
					packethandler.SetBlockAndNotify(world, falling.X, int32(falling.Y), falling.Z, &block)
				}
				continue
			}
			if !falling.Landed {
				packethandler.BroadcastRelativePosition(world, falling, prevX, prevY, prevZ, float64(falling.X), falling.Y, float64(falling.Z), 0)
			}
		}
	}
	for _, id := range toRemove {
		world.RemoveEntity(id)
		despawn := packets.EntityDespawnOutPacket{EntityId: id}
		world.BroadcastPacket(despawn.Serialize())
	}
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
			// log.Printf("TickBoat: moved to X=%.6f Y=%.6f Z=%.6f (encoded X=%d Y=%d Z=%d) yaw=%.2f",
			// 	nx, ny, nz, int32(nx), int32(ny), int32(nz), yaw)
			ridable.SetPosition(nx, ny, nz)
		case entities.Stopped:
			packethandler.BroadcastTeleport(world, ridable, cx, cy, cz, yaw)
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

		world.SetBlock(int32(x), byte(y), int32(z), newBlock)

		blockChange := packets.BlockChangeOutPacket{
			X:         int32(x),
			Y:         byte(y),
			Z:         int32(z),
			BlockType: newBlock.TypeId,
			BlockMeta: newBlock.Metadata,
		}
		world.BroadcastPacket(blockChange.Serialize())
	}
}

func furnaceLogic(world *level.World) {
	furnaces := world.GetAllFurnaces()
	inventory.TickFurnaces(furnaces, makeSendFurnaceProgress(world), makeSendFurnaceSlot(world), makeSetFurnaceBlock(world))
}
