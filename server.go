package main

import (
	"bufio"
	"log"
	"net"
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

	log.Println("Server listening on " + CON_HOST + ":" + CON_PORT)

	world := level.NewWorld(level.Stones)
	if err := world.LoadChanges(worldSavePath); err != nil {
		log.Println("Failed to load world save:", err)
	}
	if err := inventory.LoadContainers(containerSavePath); err != nil {
		log.Println("Failed to load container save:", err)
	}
	gameLoop(world)

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
		//packethandler.SetBlockAndNotify
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			// For fast time, set it to + 20
			world.Tick = (world.Tick + 1) % 24000
			world.BroadcastTime()
			minecartPhysics(world)
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
			// Save world every 1200 ticks = every 60s
			if world.Tick%1200 == 0 {
				if err := world.SaveChanges(worldSavePath); err != nil {
					log.Println("Failed to save world:", err)
				}
				if err := inventory.SaveContainers(containerSavePath); err != nil {
					log.Println("Failed to save containers:", err)
				}
			}
		}
	}()
}

func minecartPhysics(world *level.World) {
	allEntities := world.SnapshotEntities()
	var carts []*entities.RideableEntity
	var players []entities.PlayerPosition

	for _, e := range allEntities {
		if e.IsPlayer() {
			x, y, z := e.GetPosition()
			players = append(players, entities.PlayerPosition{X: x, Y: y, Z: z})
		} else if cart, ok := e.(*entities.RideableEntity); ok {
			if cart.ObjectType == 10 {
				carts = append(carts, cart)
			}
		}
	}

	getBlock := func(x int32, y byte, z int32) entities.BlockInfo {
		b := world.GetBlock(x, y, z)
		return entities.BlockInfo{
			IsRail:        b.IsRail(),
			IsPoweredRail: b.IsPoweredRail(),
			Metadata:      int(b.Metadata),
		}
	}

	var toRemove []int32

	for _, cart := range carts {
		cx, cy, cz := cart.GetPosition()
		nx, ny, nz, yaw, action := cart.TickPhysics(getBlock, players)
		switch action {
		case entities.CartMoved:
			packethandler.BroadcastRelativePosition(world, cart, cx, cy, cz, nx, ny, nz, yaw)
			cart.SetPosition(nx, ny, nz)
		case entities.CartStopped:
			packethandler.BroadcastTeleport(world, cart, cx, cy, cz, yaw)
		case entities.CartDespawned:
			despawn := packets.EntityDespawnOutPacket{EntityId: cart.EntityId}
			world.BroadcastPacket(despawn.Serialize())
			toRemove = append(toRemove, cart.EntityId)
		}
	}

	for _, id := range toRemove {
		world.RemoveEntity(id)
	}
}
