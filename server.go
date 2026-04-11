package main

import (
	"bufio"
	"log"
	"math"
	"net"
	"time"

	"github.com/leNicDev/retromc/constants"
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

func startGameLoop(world *level.World) {
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			// For fast time, set it to + 20
			world.Tick = (world.Tick + 1) % 24000
			world.BroadcastTime()
			tickPhysics(world)
		}
	}()
}

func tickPhysics(world *level.World) {
	const (
		pushRadius = 1.25
		pushForce  = 0.3
		friction   = 0.75
	)

	entities := world.SnapshotEntities()

	type playerPos struct {
		x, z float64
	}
	var players []playerPos
	var carts []*level.RideableEntity

	for _, e := range entities {
		if e.IsPlayer() {
			x, _, z := e.GetPosition()
			players = append(players, playerPos{x, z})
		} else if cart, ok := e.(*level.RideableEntity); ok {
			carts = append(carts, cart)
		}
	}

	for _, cart := range carts {
		cx, cy, cz := cart.GetPosition()
		prevVx, prevVz := cart.VelocityX, cart.VelocityZ

		for _, pp := range players {
			dx := cx - pp.x
			dz := cz - pp.z
			dist := math.Sqrt(dx*dx + dz*dz)
			if dist < pushRadius && dist > 0.001 {
				nx := dx / dist
				nz := dz / dist
				dot := nx*cart.VelocityX + nz*cart.VelocityZ
				if dot >= 0 {
					cart.VelocityX += nx * pushForce
					cart.VelocityZ += nz * pushForce
				}
			}
		}

		cart.VelocityX *= friction
		cart.VelocityZ *= friction

		if math.Abs(cart.VelocityX) < 0.001 {
			cart.VelocityX = 0
		}
		if math.Abs(cart.VelocityZ) < 0.001 {
			cart.VelocityZ = 0
		}

		railIds := map[byte]bool{
			byte(constants.Rail.Value):         true,
			byte(constants.PoweredRail.Value):  true,
			byte(constants.DetectorRail.Value): true,
		}

		railInX := func(testX float64) bool {
			bInside := world.GetBlock(int32(math.Floor(testX)), byte(math.Floor(cy)), int32(math.Floor(cz)))
			bBelow := world.GetBlock(int32(math.Floor(testX)), byte(math.Floor(cy-1)), int32(math.Floor(cz)))
			return railIds[bInside.TypeId] || railIds[bBelow.TypeId]
		}
		railInZ := func(testZ float64) bool {
			bInside := world.GetBlock(int32(math.Floor(cx)), byte(math.Floor(cy)), int32(math.Floor(testZ)))
			bBelow := world.GetBlock(int32(math.Floor(cx)), byte(math.Floor(cy-1)), int32(math.Floor(testZ)))
			return railIds[bInside.TypeId] || railIds[bBelow.TypeId]
		}

		if cart.VelocityX != 0 && !railInX(cx+cart.VelocityX) {
			cart.VelocityX = 0
		}
		if cart.VelocityZ != 0 && !railInZ(cz+cart.VelocityZ) {
			cart.VelocityZ = 0
		}

		nextX := cx + cart.VelocityX
		nextZ := cz + cart.VelocityZ

		blockBelow := world.GetBlock(int32(math.Floor(nextX)), byte(math.Floor(cy-1)), int32(math.Floor(nextZ)))
		blockInside := world.GetBlock(int32(math.Floor(nextX)), byte(math.Floor(cy)), int32(math.Floor(nextZ)))
		hasRail := railIds[blockInside.TypeId] || railIds[blockBelow.TypeId]

		if !hasRail {
			cart.VelocityX = 0
			cart.VelocityZ = 0
			nextX = cx
			nextZ = cz
		}

		if cart.VelocityX != 0 {
			wallX := world.GetBlock(int32(math.Floor(nextX)), byte(math.Floor(cy)), int32(math.Floor(cz)))
			if !wallX.IsAir() && !railIds[wallX.TypeId] {
				cart.VelocityX = 0
				nextX = cx
			}
		}
		if cart.VelocityZ != 0 {
			wallZ := world.GetBlock(int32(math.Floor(cx)), byte(math.Floor(cy)), int32(math.Floor(nextZ)))
			if !wallZ.IsAir() && !railIds[wallZ.TypeId] {
				cart.VelocityZ = 0
				nextZ = cz
			}
		}

		// Derive Y from the rail block at the current position
		bx := int32(math.Floor(cx))
		by := byte(math.Floor(cy))
		bz := int32(math.Floor(cz))

		if railIds[world.GetBlock(bx, by-1, bz).TypeId] {
			by--
		}

		// Powered rail logic
		// TODO: Something is off, boost should be lower somehow...
		currentRail := world.GetBlock(bx, by, bz)
		if currentRail.TypeId == byte(constants.PoweredRail.Value) {
			log.Println("Powered Rail Hit")
			speed := math.Sqrt(cart.VelocityX*cart.VelocityX + cart.VelocityZ*cart.VelocityZ)
			if speed > 0.01 {
				const boost = 0.5
				cart.VelocityX += cart.VelocityX / speed * boost
				cart.VelocityZ += cart.VelocityZ / speed * boost
			}
		}

		railBlock := world.GetBlock(bx, by, bz)
		trackY := float64(by)
		if railIds[railBlock.TypeId] && railBlock.Metadata >= 2 && railBlock.Metadata <= 5 {
			// Ascending rail: Y is always blockY+1, same as vanilla's trackY = blockY + 1
			trackY = float64(by) + 1.0
		}
		nextY := trackY + 0.125 // standingEyeHeight offset, equivalent to BetaSharp's setTrackAlignedPosition

		// Capture fixed-point delta before mutating position
		oldFX := int32(math.Floor(cx * 32))
		oldFY := int32(math.Floor(cy * 32))
		oldFZ := int32(math.Floor(cz * 32))
		cart.SetPosition(nextX, nextY, nextZ)
		newFX := int32(math.Floor(nextX * 32))
		newFY := int32(math.Floor(nextY * 32))
		newFZ := int32(math.Floor(nextZ * 32))

		if cart.VelocityX != prevVx || cart.VelocityZ != prevVz {
			p := packets.EntityVelocity{
				EntityId: cart.EntityId,
				Vx:       int16(cart.VelocityX),
				Vy:       0,
				Vz:       int16(cart.VelocityZ),
			}
			world.BroadcastPacket(p.Serialize())
		}

		dX := newFX - oldFX
		dY := newFY - oldFY
		dZ := newFZ - oldFZ
		if dX != 0 || dY != 0 || dZ != 0 {
			// TODO: Logic still not clean; if y-position desync, breaks...
			log.Printf("Cart Position X=%.2f, Y=%.2f Z=%.2f", nextX, nextY, nextZ)
			pkt := packets.EntityPositionOutPacket{
				EntityId: cart.EntityId,
				X:        byte(int8(dX)),
				Y:        byte(int8(dY)),
				Z:        byte(int8(dZ)),
			}
			world.BroadcastPacket(pkt.Serialize())
		}
	}
}
