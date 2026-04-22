package main

import (
	"bufio"
	"log"
	"math"
	"net"
	"time"

	"github.com/leNicDev/retromc/entities"
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

	world := level.NewWorld()
	if err := world.LoadChanges("world.dat"); err != nil {
		log.Println("Failed to load world save:", err)
	}
	startGameLoop(world)

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
			world.BroadcastPacket(packets.PlayerEntityDespawnPacket(pl))
			world.RemovePlayer(pl)
			close(done)
			connection.Close()
			return
		}
	}
}

const worldSavePath = "world.dat"

func startGameLoop(world *level.World) {
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			// For fast time, set it to + 20
			world.Tick = (world.Tick + 1) % 24000
			world.BroadcastTime()
			minecartPhysics(world)

			// Save world every 1200 ticks = every 60s
			if world.Tick%1200 == 0 {
				if err := world.SaveChanges(worldSavePath); err != nil {
					log.Println("Failed to save world:", err)
				}
			}
		}
	}()
}

// Use teleport packet to obtain absolute control over minecart
// Too bad at math to get it to work with relative positions and mimicking client-side calculations...
func broadcastTeleport(w *level.World, c *entities.RideableEntity, cx, cy, cz float64, yaw byte) {
	tpkt := packets.TeleportEntity{
		EntityId: c.EntityId,
		X:        int32(math.Floor(cx * 32)),
		Y:        int32(math.Floor(cy * 32)),
		Z:        int32(math.Floor(cz * 32)),
		Yaw:      yaw,
		Pitch:    0,
	}
	w.BroadcastPacket(tpkt.Serialize())
}

func broadcastPosition(w *level.World, c *entities.RideableEntity, prevX, prevY, prevZ, nextX, nextY, nextZ float64) {
	encPrevX := int32(math.Floor(prevX * 32))
	encPrevY := int32(math.Floor(prevY * 32))
	encPrevZ := int32(math.Floor(prevZ * 32))
	encNextX := int32(math.Floor(nextX * 32))
	encNextY := int32(math.Floor(nextY * 32))
	encNextZ := int32(math.Floor(nextZ * 32))

	dX := encNextX - encPrevX
	dY := encNextY - encPrevY
	dZ := encNextZ - encPrevZ

	// delta overflow guard — fall back to teleport if moved more than 4 blocks
	if dX < -128 || dX > 127 || dY < -128 || dY > 127 || dZ < -128 || dZ > 127 {
		broadcastTeleport(w, c, nextX, nextY, nextZ, 0)
		return
	}

	p := packets.EntityPositionOutPacket{
		EntityId: c.EntityId,
		X:        byte(dX),
		Y:        byte(dY),
		Z:        byte(dZ),
	}
	w.BroadcastPacket(p.Serialize())
}

func broadcastRelativePosition(w *level.World, c *entities.RideableEntity, prevX, prevY, prevZ, nextX, nextY, nextZ float64, yaw byte) {
	encPrevX := int32(math.Floor(prevX * 32))
	encPrevY := int32(math.Floor(prevY * 32))
	encPrevZ := int32(math.Floor(prevZ * 32))
	encNextX := int32(math.Floor(nextX * 32))
	encNextY := int32(math.Floor(nextY * 32))
	encNextZ := int32(math.Floor(nextZ * 32))

	dX := encNextX - encPrevX
	dY := encNextY - encPrevY
	dZ := encNextZ - encPrevZ

	// delta overflow guard — fall back to teleport if moved more than 4 blocks
	if dX < -128 || dX > 127 || dY < -128 || dY > 127 || dZ < -128 || dZ > 127 {
		broadcastTeleport(w, c, nextX, nextY, nextZ, 0)
		return
	}

	p := packets.EntityPositionAndLookOutPacket{
		EntityId: c.EntityId,
		X:        byte(dX),
		Y:        byte(dY),
		Z:        byte(dZ),
		Yaw:      yaw,
		Pitch:    0,
	}
	w.BroadcastPacket(p.Serialize())
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
			carts = append(carts, cart)
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
			cart.SetPosition(nx, ny, nz)
			broadcastRelativePosition(world, cart, cx, cy, cz, nx, ny, nz, yaw)
			log.Printf("Cart %d moved to (%.2f, %.2f, %.2f)", cart.EntityId, nx, ny, nz)
		case entities.CartStopped:
			broadcastTeleport(world, cart, cx, cy, cz, yaw)
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
