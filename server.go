package main

import (
	"bufio"
	"flag"
	"log"
	"net"
	"os"
	"time"

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

	// Give world access to packethandler functions due to forbidden import cycles
	world.SetBroadcastRelativePosition(packethandler.BroadcastRelativePosition)
	world.SetBroadcastEntityVelocity(packethandler.BroadcastEntityVelocity)
	world.SetCollectItem(packets.CollectItem)
	world.SetSendSetSlot(packethandler.SendSetSlot)
	world.SetBroadcastDespawn(packethandler.BroadcastDespawn)
	world.SetBroadcastTeleport(packethandler.BroadcastTeleport)
	world.SetBroadcastSetSlot(packethandler.BroadcastSetSlot)
	world.SetBroadcastContainerData(packethandler.BroadcastContainerData)
	world.SetBroadcastBlockChange(packets.BroadcastBlockChange)
	world.SetBroadcastMultiBlockChange(packets.BroadcastMultiBlockChange)
	world.SetBroadcastSpawnObject(packethandler.BroadcastSpawnObject)
	world.SetBroadcastTime(packethandler.BroadcastTime)

	entityTracker := level.NewEntityTracker(packets.SpawnPlayerEntityPacket, packets.SpawnObjectPacket, packets.EntityDespawnPacket, packets.SetEquipment2)
	gameLoop(world, entityTracker)
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	for {
		connection, err := l.Accept()
		if err != nil {
			log.Fatalln("Failed to accept connection: ", err.Error())
			continue
		}
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
					world.BroadcastPacket(packets.PlayerEntityDespawnPacket(pl))
					world.RemovePlayer(pl)
					tracker.Remove(pl.GetEntityId())
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
			nextTick := (world.Tick + world.TickSpeed) % 24000
			world.AdvanceTick(nextTick, entityTracker)
			if world.Tick%300 == 0 {
				if removed := world.PopUnusedChunks(); len(removed) > 0 {
					if err := level.SaveChunks(world, world.WorldDir, removed); err != nil {
						log.Println("Failed to save the world:", err)
					}
				}
			}
			entityTracker.Manage(world)
			world.FlushBlockQueue()
		}
	}()
}
