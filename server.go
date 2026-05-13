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

	world := level.NewWorld()
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

type FluidConfig struct {
	IsFluid         func(b level.Block) bool
	NewBlock        func(liquidHeight byte) level.Block
	Sources         map[level.BlockKey]byte
	Flowing         map[level.BlockKey]byte
	MaxSpreadHeight byte
}

func newWaterConfig(world *level.World) FluidConfig {
	return FluidConfig{
		IsFluid: func(b level.Block) bool { return b.IsWater() },
		NewBlock: func(h byte) level.Block {
			block := level.NewFlowingWaterBlock(h)
			return block
		},
		Sources:         world.WaterSources,
		Flowing:         world.FlowingWater,
		MaxSpreadHeight: 7,
	}
}

func newLavaConfig(world *level.World) FluidConfig {
	return FluidConfig{
		IsFluid: func(b level.Block) bool { return b.IsLava() },
		NewBlock: func(h byte) level.Block {
			block := level.NewFlowingLavaBlock(h)
			return block
		},
		Sources:         world.LavaSources,
		Flowing:         world.FlowingLava,
		MaxSpreadHeight: 7,
	}
}

func setFlowingFluid(world *level.World, x, y, z int32, liquidHeight byte, cfg FluidConfig) {
	block := cfg.NewBlock(liquidHeight)
	packethandler.SetBlockAndNotify(world, x, y, z, &block)
	key := level.BlockKey{X: x, Y: byte(y), Z: z}
	cfg.Flowing[key] = liquidHeight
}

func fluidDecay(world *level.World, cfg FluidConfig) {
	visited := make(map[level.BlockKey]bool)
	queue := []level.BlockKey{}

	for key := range cfg.Sources {
		// NOTE: without this check, decay breaks
		if _, isFlowing := cfg.Flowing[key]; !isFlowing {
			visited[key] = true
			queue = append(queue, key)
		}
	}

	spreadNeighbors := []struct{ dx, dy, dz int32 }{
		{1, 0, 0}, {-1, 0, 0}, {0, 0, 1}, {0, 0, -1}, {0, -1, 0},
	}
	// BFS to find all possible flowing blocks connected to sources
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		x, y, z := key.X, key.Y, key.Z

		for _, n := range spreadNeighbors {
			nx := x + n.dx
			ny := int32(y) + n.dy
			nz := z + n.dz
			if ny < 0 || ny > 255 {
				continue
			}
			nKey := level.BlockKey{X: nx, Y: byte(ny), Z: nz}
			if visited[nKey] {
				continue
			}
			if cfg.IsFluid(world.GetBlock(nx, byte(ny), nz)) {
				visited[nKey] = true
				queue = append(queue, nKey)
			}
		}
	}

	for key := range cfg.Flowing {
		if !visited[key] {
			air := level.NewAirBlock()
			packethandler.SetBlockAndNotify(world, key.X, int32(key.Y), key.Z, &air)
			delete(cfg.Flowing, key)
		}
	}
}

func abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

func findHoleNearSource(world *level.World, sourceKey level.BlockKey, cfg FluidConfig) (level.BlockKey, bool) {
	const maxDist = 4
	x, y, z := sourceKey.X, sourceKey.Y, sourceKey.Z

	visited := make(map[level.BlockKey]bool)
	queue := []level.BlockKey{{X: x, Y: y, Z: z}}
	visited[sourceKey] = true

	// BFS to find the nearest reachable hole
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		b := world.GetBlock(cur.X, cur.Y-1, cur.Z)
		// a hole must be either air or flowing water
		// latter allows us to keep the same state even if the hole is filled with flowing water
		if cur.Y > 0 && (b.IsAir() || b.IsFLowing()) {
			return cur, true
		}

		if abs(cur.X-x) >= int32(maxDist) || abs(cur.Z-z) >= int32(maxDist) {
			continue
		}

		for _, n := range []struct{ dx, dz int32 }{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, nz := cur.X+n.dx, cur.Z+n.dz
			nKey := level.BlockKey{X: nx, Y: y, Z: nz}
			if visited[nKey] {
				continue
			}
			visited[nKey] = true
			if b := world.GetBlock(nx, y, nz); b.IsAir() || cfg.IsFluid(b) {
				queue = append(queue, nKey)
			}
		}
	}
	return level.BlockKey{}, false
}

func fluidSpreading(world *level.World, cfg FluidConfig) {
	type fluidEntry struct {
		key    level.BlockKey
		height byte
	}
	var allFluids []fluidEntry
	for key, h := range cfg.Sources {
		allFluids = append(allFluids, fluidEntry{key, h})
	}

	for _, entry := range allFluids {
		x, y, z := entry.key.X, entry.key.Y, entry.key.Z
		height := entry.height

		b := world.GetBlock(x, y-1, z)
		if y > 0 && b.IsAir() {
			setFlowingFluid(world, x, int32(y)-1, z, 0, cfg)
			continue
		}

		if height >= cfg.MaxSpreadHeight {
			continue
		}
		nextHeight := height + 1

		type offset struct{ dx, dz int32 }
		neighbors := []offset{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
		holeTarget, holeFound := findHoleNearSource(world, entry.key, cfg)
		if holeFound {
			var biased []offset
			for _, n := range neighbors {
				nx, nz := x+n.dx, z+n.dz
				curDist := abs(x-holeTarget.X) + abs(z-holeTarget.Z)
				newDist := abs(nx-holeTarget.X) + abs(nz-holeTarget.Z)
				if newDist < curDist {
					biased = append(biased, n)
				}
			}
			neighbors = biased
			if len(neighbors) == 0 {
				continue
			}
		}

		for _, n := range neighbors {
			nx, nz := x+n.dx, z+n.dz
			b := world.GetBlock(nx, y, nz)
			if !b.IsAir() {
				continue
			}
			nKey := level.BlockKey{X: nx, Y: y, Z: nz}
			if existing, exists := cfg.Flowing[nKey]; exists && existing <= nextHeight {
				continue
			}
			setFlowingFluid(world, nx, int32(y), nz, nextHeight, cfg)
		}
	}
}

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
			if world.Tick%20 == 0 || world.Tick%60 == 0 {
				waterCfg := newWaterConfig(world)
				lavaCfg := newLavaConfig(world)
				fluidDecay(world, waterCfg)
				fluidSpreading(world, waterCfg)
				fluidDecay(world, lavaCfg)
				if world.Tick%60 == 0 {
					fluidSpreading(world, lavaCfg)
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
