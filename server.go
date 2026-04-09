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

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func tickPhysics(world *level.World) {
	const (
		pushRadius = 1.25
		pushForce  = 0.3
		friction   = 0.98
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

		// Snap Y to rail surface every tick
		snapY := math.Floor(cy) + 0.125
		if snapY != cy {
			cart.SetPosition(cx, snapY, cz)
			cy = snapY
		}

		prevVx, prevVz := cart.VelocityX, cart.VelocityZ

		for _, pp := range players {
			dx := cx - pp.x
			dz := cz - pp.z
			dist := math.Sqrt(dx*dx + dz*dz)
			if dist < pushRadius && dist > 0.001 {
				// Stronger single impulse rather than tiny per-tick accumulation
				nx := dx / dist
				nz := dz / dist
				// Only push if player is actually moving toward the cart
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

		cart.VelocityX = clamp(cart.VelocityX, -3.9, 3.9)
		cart.VelocityZ = clamp(cart.VelocityZ, -3.9, 3.9)

		nextX := cx + cart.VelocityX
		nextZ := cz + cart.VelocityZ

		railIds := map[byte]bool{
			byte(constants.Rail.Value):         true,
			byte(constants.PoweredRail.Value):  true,
			byte(constants.DetectorRail.Value): true,
		}

		blockBelow := world.GetBlock(int32(math.Floor(nextX)), byte(math.Floor(cy-1)), int32(math.Floor(nextZ)))
		blockAtFeet := world.GetBlock(int32(math.Floor(nextX)), byte(math.Floor(cy)), int32(math.Floor(nextZ)))
		hasRail := railIds[blockAtFeet.TypeId] || railIds[blockBelow.TypeId]

		if !hasRail {
			cart.VelocityX = 0
			cart.VelocityZ = 0
			nextX = cx
			nextZ = cz
		}

		// Capture fixed-point delta before mutating position
		oldFX := int32(math.Floor(cx * 32))
		oldFZ := int32(math.Floor(cz * 32))
		cart.SetPosition(nextX, cy, nextZ)
		newFX := int32(math.Floor(nextX * 32))
		newFZ := int32(math.Floor(nextZ * 32))

		// Only send velocity when it meaningfully changed
		if cart.VelocityX != prevVx || cart.VelocityZ != prevVz {
			p := packets.EntityVelocity{
				EntityId: cart.EntityId,
				Vx:       int16(cart.VelocityX * 8000),
				Vy:       0,
				Vz:       int16(cart.VelocityZ * 8000),
			}
			world.BroadcastPacket(p.Serialize())
		}

		dX := newFX - oldFX
		dZ := newFZ - oldFZ
		if dX != 0 || dZ != 0 {
			pkt := packets.EntityPositionOutPacket{
				EntityId: cart.EntityId,
				X:        byte(int8(dX)),
				Y:        0,
				Z:        byte(int8(dZ)),
			}
			world.BroadcastPacket(pkt.Serialize())
		}
	}
}
