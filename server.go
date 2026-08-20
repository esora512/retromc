package main

import (
	"bufio"
	"flag"
	"log"
	"net"
	"os"
	"strings"
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

type opUsernamesFlag map[string]bool

func (o opUsernamesFlag) String() string {
	names := make([]string, 0, len(o))
	for name := range o {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

func (o opUsernamesFlag) Set(value string) error {
	o[strings.ToLower(value)] = true
	return nil
}

func main() {
	host := flag.String("host", "localhost", "Address to bind the server to")
	port := flag.String("port", "25565", "Port to bind the server to")

	ops := make(opUsernamesFlag)
	flag.Var(&ops, "op", "Username of a player to grant operator permissions to (repeatable, e.g. --op esora512 --op PixelBrush)")

	flag.Parse()
	l, err := net.Listen(CON_TYPE, *host+":"+*port)
	if err != nil {
		log.Panicln("Failed to bind to address", err.Error())
	}

	// close listener when the application closes
	defer l.Close()

	log.Printf("Server listening on %s:%s (PID: %d)", *host, *port, os.Getpid())

	world := level.NewWorld(GitCommit, 0, level.Template)

	// Give world access to packethandler functions due to forbidden import cycles
	world.SetNewPositionAndRotationOrTeleportPacket(packethandler.NewPositionAndRotationOrTeleportPacket)
	world.SetNewTeleportPacket(packethandler.NewTeleportPacket)
	world.SetNewPositionPacket(packethandler.NewPositionOrTeleportPacket)
	world.SetNewRotationPacket(packethandler.NewRotationPacket)

	world.SetBroadcastPositionAndRotation(packethandler.BroadcastPositionAndRotation)
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
	world.SetBroadcastWakeUp(packethandler.BroadcastWakeUp)
	world.SetBroadcastWorldMsg(packethandler.BroadcastWorldMsg)
	world.SetBroadcastMobSpawn(packets.BroadcastMobSpawn)
	world.SetBroadcastMobPositionAndRotation(packets.BroadcastMobPositionAndRotation)
	world.SetSendSetHealth(packethandler.SendSetHealth)
	world.SetBroadcastPain(packethandler.BroadcastPain)
	world.SetNewMobPositionAndRotationPacket(packets.NewMobPositionAndRotationPacket)
	world.SetNewEntityVelocityPacket(packethandler.NewEntityVelocityPacket)
	world.SetAndCreateAndSetMovementDroppedItem(packethandler.CreateAndSetMovementDroppedItem)

	world.SetOppedUsernames(ops)

	entityTracker := level.NewEntityTracker(packets.NewSpawnPlayerPacket, 
											packets.NewSpawnObjectPacket, 
											packets.SpawnMob, 
											packets.NewSpawnItem, 
											packets.NewEntityDespawnPacket, 
											packets.SetEquipment2)


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
		keepAlive := packets.KeepAlivePacket{}
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
					world.BroadcastPacket(packets.NewEntityDespawnPacket(pl.GetEntityId()))
					world.RemovePlayer(pl)
					tracker.ResetEntity(pl.GetEntityId())
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
				if removed := world.PopUnusedChunks(0); len(removed) > 0 {
					if err := level.SaveChunks(world, world.WorldDir, removed, 0); err != nil {
						log.Println("Failed to save the world:", err)
					}
				}
				if removed := world.PopUnusedChunks(-1); len(removed) > 0 {
					if err := level.SaveChunks(world, world.WorldDir, removed, -1); err != nil {
						log.Println("Failed to save the world:", err)
					}
				}
			}
			entityTracker.Manage(world)
			world.FlushBlockQueue()
		}
	}()
}
