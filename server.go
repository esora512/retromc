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

	// findRail returns the Y of the rail block at (bx, by, bz),
	// also check one block below to handle the cart sitting above the rail.
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

		// Find rail at the cart's current block position.
		bx := int32(math.Floor(cx))
		by := byte(math.Floor(cy))
		bz := int32(math.Floor(cz))

		railY, hasRail := findRail(bx, by, bz)
		if !hasRail {
			// Destroy cart if no rail found (e.g. rail destroyed while cart on it).
			despawn := packets.EntityDespawnOutPacket{EntityId: cart.EntityId}
			world.BroadcastPacket(despawn.Serialize())
			toRemove = append(toRemove, cart.EntityId)
			continue
		}
		by = railY

		railBlock := world.GetBlock(bx, by, bz)
		rawMeta := railBlock.Metadata
		// Powered/detector rails store power state in bit 3; strip it for direction.
		meta := rawMeta
		if railBlock.TypeId == byte(constants.PoweredRail.Value) ||
			railBlock.TypeId == byte(constants.DetectorRail.Value) {
			meta &= 0x07
		}

		prevVx, prevVz := cart.VelocityX, cart.VelocityZ

		// Apply player push forces.
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

		// Powered rail boost: vanilla uses 0.06 per tick when powered
		if railBlock.TypeId == byte(constants.PoweredRail.Value) {
			speed := math.Sqrt(cart.VelocityX*cart.VelocityX + cart.VelocityZ*cart.VelocityZ)
			if speed > 0.01 {
				const boost = 0.06 * 10
				cart.VelocityX += cart.VelocityX / speed * boost
				cart.VelocityZ += cart.VelocityZ / speed * boost
			}
		}

		// Constrain velocity to the rail's axis, then apply friction.
		// meta 0: flat N-S (Z axis)  → zero X velocity
		// meta 1: flat E-W (X axis)  → zero Z velocity
		// meta 2,3: ascending E-W    → zero Z velocity
		// meta 4,5: ascending N-S    → zero X velocity
		// meta 6-9: curves           → allow both axes
		switch meta {
		case 0, 4, 5:
			cart.VelocityX = 0
		case 1, 2, 3:
			cart.VelocityZ = 0
		}

		cart.VelocityX *= friction
		cart.VelocityZ *= friction
		if math.Abs(cart.VelocityX) < 0.001 {
			cart.VelocityX = 0
		}
		if math.Abs(cart.VelocityZ) < 0.001 {
			cart.VelocityZ = 0
		}

		// Compute candidate next position, snapping the perpendicular axis to
		// the rail block's centre (keeps the cart on the track).
		var nextX, nextZ float64
		switch meta {
		case 0, 4, 5: // N-S and ascending N-S
			nextX = float64(bx) + 0.5 // snap X to rail centre
			nextZ = cz + cart.VelocityZ
		case 1, 2, 3: // E-W and ascending E-W
			nextZ = float64(bz) + 0.5 // snap Z to rail centre
			nextX = cx + cart.VelocityX
		default: // curves
			nextX = cx + cart.VelocityX
			nextZ = cz + cart.VelocityZ
		}

		// If the candidate block differs, verify a rail exists there (same Y,
		// one above for ascending exit, or one below for descending entry).
		nextBx := int32(math.Floor(nextX))
		nextBz := int32(math.Floor(nextZ))
		if nextBx != bx || nextBz != bz {
			_, okSame := findRail(nextBx, by, nextBz)
			_, okAbove := findRail(nextBx, by+1, nextBz)
			if !okSame && !okAbove {
				// No rail in the next block — stop the cart at the current rail centre.
				cart.VelocityX = 0
				cart.VelocityZ = 0
				nextX = float64(bx) + 0.5
				nextZ = float64(bz) + 0.5
			}
		}

		// Derive Y from the rail geometry at the *destination* block.
		// We must re-look-up the rail at (floor(nextX), floor(nextZ)) because when
		// the cart crosses a block boundary the destination rail may be at a
		// different Y than the current one (e.g. ascending → flat one level up).
		// Using the current block's `by`/`meta` for a position in the next block
		// produces the wrong Y, causing findRail to fail on the following tick and
		// the cart to be incorrectly despawned.
		destBx := int32(math.Floor(nextX))
		destBz := int32(math.Floor(nextZ))
		destRailY := by // default: same level as current rail
		if !railIds[world.GetBlock(destBx, by, destBz).TypeId] {
			if by > 0 && railIds[world.GetBlock(destBx, by-1, destBz).TypeId] {
				destRailY = by - 1 // descending into lower block
			} else if railIds[world.GetBlock(destBx, by+1, destBz).TypeId] {
				destRailY = by + 1 // ascending into higher block
			}
		}
		destBlock := world.GetBlock(destBx, destRailY, destBz)
		destMeta := destBlock.Metadata
		if destBlock.TypeId == byte(constants.PoweredRail.Value) ||
			destBlock.TypeId == byte(constants.DetectorRail.Value) {
			destMeta &= 0x07
		}

		var nextY float64
		switch destMeta {
		case 2: // ascending east (+X rises)
			t := nextX - math.Floor(nextX)
			nextY = float64(destRailY) + 0.5 + t
		case 3: // ascending west (-X rises)
			t := nextX - math.Floor(nextX)
			nextY = float64(destRailY) + 1.5 - t
		case 4: // ascending north (-Z rises)
			t := nextZ - math.Floor(nextZ)
			nextY = float64(destRailY) + 1.5 - t
		case 5: // ascending south (+Z rises)
			t := nextZ - math.Floor(nextZ)
			nextY = float64(destRailY) + 0.5 + t
		default: // flat (meta 0, 1, 6-9)
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

		// TODO:
		// There is a desync issue involving the Y-positions and I could not figure out why
		// Thus, instead of using EntityPosition (relative), I use TeleportEntity (absolute)
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
