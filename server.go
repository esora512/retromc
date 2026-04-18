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

var railDirs = [10][2][3]int{
	0: {{0, 0, -1}, {0, 0, 1}},
	1: {{-1, 0, 0}, {1, 0, 0}},
	2: {{-1, -1, 0}, {1, 0, 0}},
	3: {{-1, 0, 0}, {1, -1, 0}},
	4: {{0, 0, -1}, {0, -1, 1}},
	5: {{0, -1, -1}, {0, 0, 1}},
	6: {{0, 0, 1}, {1, 0, 0}},
	7: {{0, 0, 1}, {-1, 0, 0}},
	8: {{0, 0, -1}, {-1, 0, 0}},
	9: {{0, 0, -1}, {1, 0, 0}},
}

// getRailPos mirrors func_182_g — returns the interpolated position
// on the rail curve/slope for a given world position.
func getRailPos(world *level.World, px, py, pz float64) (float64, float64, float64, bool) {
	bx := int32(math.Floor(px))
	by := int32(math.Floor(py))
	bz := int32(math.Floor(pz))

	block := world.GetBlock(bx, byte(by), bz)
	if !block.IsRail() {
		block = world.GetBlock(bx, byte(by-1), bz)
		if !block.IsRail() {
			return 0, 0, 0, false
		}
		by--
	}

	meta := int(block.Metadata)
	if block.IsPoweredRail() {
		meta &= 7
	}

	// //fy := float64(by)
	// if meta >= 2 && meta <= 5 {
	// 	fy := float64(by + 1)
	// }

	dirs := railDirs[meta]
	// endpoints of the rail segment in world space
	x1 := float64(bx) + 0.5 + float64(dirs[0][0])*0.5
	y1 := float64(by) + 0.5 + float64(dirs[0][1])*0.5
	z1 := float64(bz) + 0.5 + float64(dirs[0][2])*0.5
	x2 := float64(bx) + 0.5 + float64(dirs[1][0])*0.5
	y2 := float64(by) + 0.5 + float64(dirs[1][1])*0.5
	z2 := float64(bz) + 0.5 + float64(dirs[1][2])*0.5

	dx := x2 - x1
	dy := (y2 - y1) * 2.0 // doubled like the original
	dz := z2 - z1

	var t float64
	if dx == 0.0 {
		px = float64(bx) + 0.5
		t = pz - float64(bz)
	} else if dz == 0.0 {
		pz = float64(bz) + 0.5
		t = px - float64(bx)
	} else {
		t = ((px-x1)*dx + (pz-z1)*dz) * 2.0
	}

	rx := x1 + dx*t
	ry := y1 + dy*t
	rz := z1 + dz*t

	// original adjusts Y based on dy direction
	if dy < 0.0 {
		ry += 1.0
	}
	if dy > 0.0 {
		ry += 0.5
	}

	return rx, ry, rz, true
}

// Use teleport packet to obtain absolute control over minecart
// Too bad at math to get it to work with relative positions and mimicking client-side calculations...
func broadcastTeleport(w *level.World, c *entities.RideableEntity, cx, cy, cz float64) {
	tpkt := packets.TeleportEntity{
		EntityId: c.EntityId,
		X:        int32(math.Floor(cx * 32)),
		Y:        int32(math.Floor(cy * 32)),
		Z:        int32(math.Floor(cz * 32)),
		Yaw:      0,
		Pitch:    0,
	}
	w.BroadcastPacket(tpkt.Serialize())
}

func minecartPhysics(world *level.World) {
	const maxSpeed = 0.4

	allEntities := world.SnapshotEntities()
	var carts []*entities.RideableEntity

	type playerPos struct{ x, z float64 }
	var players []playerPos

	for _, e := range allEntities {
		if e.IsPlayer() {
			x, _, z := e.GetPosition()
			players = append(players, playerPos{x, z})
		} else if cart, ok := e.(*entities.RideableEntity); ok {
			carts = append(carts, cart)
		}
	}

	var toRemove []int32

	for _, cart := range carts {
		cx, cy, cz := cart.GetPosition()

		bx := int32(math.Floor(cx))
		by := int32(math.Floor(cy))
		bz := int32(math.Floor(cz))

		// check one below, like the original
		block := world.GetBlock(bx, byte(by-1), bz)
		if block.IsRail() {
			by--
		}

		block = world.GetBlock(bx, byte(by), bz)
		// Notch off-rail behaviour
		// if !block.IsRail() {
		// 	// off-rail behaviour: gravity + air friction, no boost
		// 	cart.VelocityY -= 0.04
		// 	cart.VelocityX = clamp(cart.VelocityX, -maxSpeed, maxSpeed)
		// 	cart.VelocityZ = clamp(cart.VelocityZ, -maxSpeed, maxSpeed)
		// 	cx += cart.VelocityX
		// 	cy += cart.VelocityY
		// 	cz += cart.VelocityZ
		// 	cart.VelocityX *= 0.95
		// 	cart.VelocityY *= 0.95
		// 	cart.VelocityZ *= 0.95
		// 	cart.SetPosition(cx, cy, cz)
		// 	broadcastTeleport(world, cart, cx, cy, cz)
		// 	continue
		// }

		if !block.IsRail() {
			despawn := packets.EntityDespawnOutPacket{EntityId: cart.EntityId}
			world.BroadcastPacket(despawn.Serialize())
			toRemove = append(toRemove, cart.EntityId)
			continue
		}

		// strip powered-rail activation bit to get shape meta
		meta := int(block.Metadata)
		if block.IsPoweredRail() {
			meta &= 7
		}

		// slope gravity nudge — exact values from original
		switch meta {
		case 2:
			cart.VelocityX -= 1.0 / 128.0
		case 3:
			cart.VelocityX += 1.0 / 128.0
		case 4:
			cart.VelocityZ += 1.0 / 128.0
		case 5:
			cart.VelocityZ -= 1.0 / 128.0
		}

		// player push (keep your existing logic, it's fine)
		const pushRadius, pushForce = 1.25, 0.3
		for _, pp := range players {
			dx := cx - pp.x
			dz := cz - pp.z
			dist := math.Sqrt(dx*dx + dz*dz)
			if dist < pushRadius && dist > 0.001 {
				nx, nz := dx/dist, dz/dist
				if nx*cart.VelocityX+nz*cart.VelocityZ >= 0 {
					cart.VelocityX += nx * pushForce
					cart.VelocityZ += nz * pushForce
				}
			}
		}

		// align velocity to rail direction using the table
		dirs := railDirs[meta]
		dirX := float64(dirs[1][0] - dirs[0][0])
		dirZ := float64(dirs[1][2] - dirs[0][2])
		dirLen := math.Sqrt(dirX*dirX + dirZ*dirZ)

		dot := cart.VelocityX*dirX + cart.VelocityZ*dirZ
		if dot < 0 {
			dirX, dirZ = -dirX, -dirZ
		}

		speed := math.Sqrt(cart.VelocityX*cart.VelocityX + cart.VelocityZ*cart.VelocityZ)
		cart.VelocityX = speed * dirX / dirLen
		cart.VelocityZ = speed * dirZ / dirLen

		// powered rail: boost or brake
		if block.IsPoweredRail() {
			isActivated := true
			//isActivated := (block.Metadata & 8) != 0
			if isActivated {
				// boost
				if speed > 0.01 {
					cart.VelocityX += cart.VelocityX / speed * 0.06
					cart.VelocityZ += cart.VelocityZ / speed * 0.06
				}
			} else {
				// brake — unpowered powered rail
				if speed < 0.03 {
					cart.VelocityX = 0
					cart.VelocityY = 0
					cart.VelocityZ = 0
				} else {
					cart.VelocityX *= 0.5
					cart.VelocityY = 0
					cart.VelocityZ *= 0.5
				}
			}
		}

		// cap speed
		cart.VelocityX = clamp(cart.VelocityX, -maxSpeed, maxSpeed)
		cart.VelocityZ = clamp(cart.VelocityZ, -maxSpeed, maxSpeed)

		// move
		nextX := cx + cart.VelocityX
		nextZ := cz + cart.VelocityZ

		// get Y before and after for hill momentum transfer
		_, prevY, _, hasPrev := getRailPos(world, cx, cy, cz)
		rx, nextY, rz, hasNext := getRailPos(world, nextX, cy, nextZ)

		if hasNext {
			nextX = rx
			nextZ = rz
			// hill momentum: going downhill adds speed, uphill removes it
			if hasPrev {
				slope := (prevY - nextY) * 0.05
				speed = math.Sqrt(cart.VelocityX*cart.VelocityX + cart.VelocityZ*cart.VelocityZ)
				if speed > 0 {
					cart.VelocityX = cart.VelocityX / speed * (speed + slope)
					cart.VelocityZ = cart.VelocityZ / speed * (speed + slope)
				}
			}
		} else {
			// no rail at destination — despawn
			// despawn := packets.EntityDespawnOutPacket{EntityId: cart.EntityId}
			// world.BroadcastPacket(despawn.Serialize())
			// toRemove = append(toRemove, cart.EntityId)
			// continue

			// stop cart when it is about to go off-rails
			cart.VelocityX = 0
			cart.VelocityZ = 0
			cart.VelocityY = 0
			broadcastTeleport(world, cart, cx, cy, cz)
			continue
		}

		// friction: 0.96 unoccupied (0.997 if rider — add that check if you have passenger support)
		cart.VelocityX *= 0.96
		cart.VelocityZ *= 0.96
		cart.VelocityY = 0 // Y motion zeroed while on rail

		if math.Abs(cart.VelocityX) < 0.001 {
			cart.VelocityX = 0
		}
		if math.Abs(cart.VelocityZ) < 0.001 {
			cart.VelocityZ = 0
		}

		cart.SetPosition(nextX, nextY, nextZ)
		broadcastTeleport(world, cart, nextX, nextY, nextZ)
	}

	for _, id := range toRemove {
		world.RemoveEntity(id)
	}
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
