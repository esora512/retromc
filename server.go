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
			minecartPhysics(world)
		}
	}()
}

func minecartPhysics(world *level.World) {
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

	railIds := map[byte]bool{
		byte(constants.Rail.Value):         true,
		byte(constants.PoweredRail.Value):  true,
		byte(constants.DetectorRail.Value): true,
	}

	findRail := func(bx int32, by byte, bz int32) (byte, bool) {
		if railIds[world.GetBlock(bx, by, bz).TypeId] {
			return by, true
		}
		if by > 0 && railIds[world.GetBlock(bx, by-1, bz).TypeId] {
			return by - 1, true
		}
		return 0, false
	}

	var toRemove []int32

	for _, cart := range carts {
		cx, cy, cz := cart.GetPosition()

		bx := int32(math.Floor(cx))
		by := byte(math.Floor(cy))
		bz := int32(math.Floor(cz))

		railY, hasRail := findRail(bx, by, bz)
		if !hasRail {
			despawn := packets.EntityDespawnOutPacket{EntityId: cart.EntityId}
			world.BroadcastPacket(despawn.Serialize())
			toRemove = append(toRemove, cart.EntityId)
			continue
		}
		by = railY

		railBlock := world.GetBlock(bx, by, bz)
		rawMeta := railBlock.Metadata
		meta := rawMeta
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

		if railBlock.TypeId == byte(constants.PoweredRail.Value) {
			speed := math.Sqrt(cart.VelocityX*cart.VelocityX + cart.VelocityZ*cart.VelocityZ)
			if speed > 0.01 {
				const boost = 0.70
				cart.VelocityX += cart.VelocityX / speed * boost
				cart.VelocityZ += cart.VelocityZ / speed * boost
			}
		}

		// Constrain velocity to rail axis, distribute to both axes for curves.
		// TODO: It works but has still some edge cases where it struggles to move...
		switch meta {
		case 0, 4, 5: // flat N-S, ascending N-S → zero X
			cart.VelocityX = 0
		case 1, 2, 3: // flat E-W, ascending E-W → zero Z
			cart.VelocityZ = 0
		case 6: // xMzP: connects -X and +Z
			if cart.VelocityX == 0 {
				cart.VelocityX = -cart.VelocityZ
			} else if cart.VelocityZ == 0 {
				cart.VelocityZ = -cart.VelocityX
			}
		case 7: // xPzP: connects +X and +Z
			if cart.VelocityX == 0 {
				cart.VelocityX = cart.VelocityZ
			} else if cart.VelocityZ == 0 {
				cart.VelocityZ = cart.VelocityX
			}
		case 8: // xPzM: connects +X and -Z
			if cart.VelocityX == 0 {
				cart.VelocityX = -cart.VelocityZ
			} else if cart.VelocityZ == 0 {
				cart.VelocityZ = -cart.VelocityX
			}
		case 9: // xMzM: connects -X and -Z
			if cart.VelocityX == 0 {
				cart.VelocityX = cart.VelocityZ
			} else if cart.VelocityZ == 0 {
				cart.VelocityZ = cart.VelocityX
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

		var nextX, nextZ float64
		switch meta {
		case 0, 4, 5:
			nextX = float64(bx) + 0.5
			nextZ = cz + cart.VelocityZ
		case 1, 2, 3:
			nextZ = float64(bz) + 0.5
			nextX = cx + cart.VelocityX
		default:
			nextX = cx + cart.VelocityX
			nextZ = cz + cart.VelocityZ
		}

		nextBx := int32(math.Floor(nextX))
		nextBz := int32(math.Floor(nextZ))
		if nextBx != bx || nextBz != bz {
			_, okSame := findRail(nextBx, by, nextBz)
			_, okAbove := findRail(nextBx, by+1, nextBz)
			if !okSame && !okAbove {
				cart.VelocityX = 0
				cart.VelocityZ = 0
				nextX = float64(bx) + 0.5
				nextZ = float64(bz) + 0.5
			}
		}

		destBx := int32(math.Floor(nextX))
		destBz := int32(math.Floor(nextZ))
		destRailY := by
		if !railIds[world.GetBlock(destBx, by, destBz).TypeId] {
			if by > 0 && railIds[world.GetBlock(destBx, by-1, destBz).TypeId] {
				destRailY = by - 1
			} else if railIds[world.GetBlock(destBx, by+1, destBz).TypeId] {
				destRailY = by + 1
			}
		}
		destBlock := world.GetBlock(destBx, destRailY, destBz)
		destMeta := destBlock.Metadata

		var nextY float64
		switch destMeta {
		case 2:
			t := nextX - math.Floor(nextX)
			nextY = float64(destRailY) + 0.5 + t
		case 3:
			t := nextX - math.Floor(nextX)
			nextY = float64(destRailY) + 1.5 - t
		case 4:
			t := nextZ - math.Floor(nextZ)
			nextY = float64(destRailY) + 1.5 - t
		case 5:
			t := nextZ - math.Floor(nextZ)
			nextY = float64(destRailY) + 0.5 + t
		default:
			nextY = float64(destRailY) + 0.5
		}

		cart.SetPosition(nextX, nextY, nextZ)

		if cart.VelocityX != prevVx || cart.VelocityZ != prevVz {
			p := packets.EntityVelocity{
				EntityId: cart.EntityId,
				Vx:       int16(cart.VelocityX),
				Vy:       0,
				Vz:       int16(cart.VelocityZ),
			}
			world.BroadcastPacket(p.Serialize())
		}

		log.Printf("Minecart Position X=%.2f Y=%.2f Z=%.2f", nextX, nextY, nextZ)
		tpkt := packets.TeleportEntity{
			EntityId: cart.EntityId,
			X:        int32(math.Floor(nextX * 32)),
			Y:        int32(math.Floor(nextY * 32)),
			Z:        int32(math.Floor(nextZ * 32)),
			Yaw:      0,
			Pitch:    0,
		}
		world.BroadcastPacket(tpkt.Serialize())
	}

	for _, id := range toRemove {
		world.RemoveEntity(id)
	}
}
