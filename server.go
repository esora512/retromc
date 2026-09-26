package main

import (
	"bufio"
	"flag"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/leNicDev/retromc/entities"
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

	wType := flag.String("wt", "Default", "World type for world generation")

	externalChunkGenBin := flag.String("external-chunkgen-bin", "", "Path to an external chunk-generation binary (e.g. bin/chunkgen); if set, routes normal terrain generation through it instead of the built-in Go generator, falling back to Go on failure")

	flag.Parse()
	world := level.NewWorld(GitCommit, 3257840388504953787, level.GetWorldType(*wType))

	// Give world access to packethandler functions due to forbidden import cycles
	world.SetNewEntityEventPacket(packethandler.NewEntityEventPacket)
	world.SetNewInteractWithBlockPacket(packets.NewInteractWithBlockPacket)

	world.SetNewMobPositionAndRotationOrTeleportPacket(packets.NewMobPositionAndRotationPacketV2)
	world.SetNewPositionAndRotationOrTeleportPacket(packethandler.NewPositionAndRotationOrTeleportPacket)
	world.SetNewTeleportPacket(packethandler.NewTeleportPacket)
	world.SetNewPositionPacket(packethandler.NewPositionOrTeleportPacket)
	world.SetNewRotationPacket(packethandler.NewRotationPacket)
	world.SetNewEntityVelocityPacket(packethandler.NewEntityVelocityPacket)

	world.SetCollectItem(packets.CollectItem)
	world.SetSendSetSlot(packethandler.SendSetSlot)
	world.SetSendContainerData(packethandler.SendContainerData)
	world.SetBroadcastSetSlot(packethandler.BroadcastSetSlot)
	world.SetBroadcastContainerData(packethandler.BroadcastContainerData)
	world.SetBroadcastBlockChange(packets.BroadcastBlockChange)
	world.SetBroadcastMultiBlockChange(packets.BroadcastMultiBlockChange)
	world.SetBroadcastTime(packethandler.BroadcastTime)
	world.SetBroadcastWorldMsg(packethandler.BroadcastWorldMsg)
	world.SetSendSetHealth(packethandler.SendSetHealth)
	world.SetHurtPlayer(packethandler.HurtPlayer)
	world.SetDropItemFromMinedBlock(packethandler.DropItemFromMinedBlock)

	world.SetOppedUsernames(ops)

	if *externalChunkGenBin != "" {
		if _, err := os.Stat(*externalChunkGenBin); err != nil {
			log.Printf("external-chunkgen-bin set to %q but not accessible (%v); will fall back to the Go generator for every chunk until this is fixed", *externalChunkGenBin, err)
		}
		world.SetExternalChunkGenBin(*externalChunkGenBin)
		log.Printf("External chunk generation enabled via %s", *externalChunkGenBin)
	}

	world.SetSpawnPlayer(packets.NewSpawnPlayerPacket)
	world.SetSpawnObject(packets.NewSpawnObjectPacket)
	world.SetSpawnMob(packets.SpawnMob)
	world.SetSpawnItem(packets.NewSpawnItem)
	world.SetSendEquipment(packets.SetEquipment3)
	world.SetDespawnEntity(packets.NewEntityDespawnPacket)

	world.SetNewAnimationPacket(packethandler.NewAnimationPacket)
	world.SetNewEntityMetadataPacket(packets.NewEntityMetadataPacket)

	entityTracker := entities.NewEntityTracker()
	server := Server{World: world, Tracker: entityTracker}
	runOnRender(&server)
	server.Run()
	startShutdownSave(world)

	l, err := net.Listen(CON_TYPE, *host+":"+*port)
	if err != nil {
		log.Panicln("Failed to bind to address", err.Error())
	}

	// close listener when the application closes
	defer l.Close()

	log.Printf("Server listening on %s:%s (PID: %d)", *host, *port, os.Getpid())

	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	for {
		connection, err := l.Accept()
		if err != nil {
			log.Fatalln("Failed to accept connection: ", err.Error())
			continue
		}
		go handleConnection(player.NewAsyncConn(connection), world, entityTracker)
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

func handleConnection(connection net.Conn, world *level.World, tracker *entities.EntityTracker) {
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
				world.Enqueue(func() {
					if cur, ok := world.GetPlayerByUsername(pl.Username); !ok || cur == pl {
						world.SavePlayer(pl)
						if pl.LoggedIn {
							p := packethandler.NewLeftGameMsg(pl.Username)
							world.BroadcastPacket(p)
							world.BroadcastPacket(packets.NewEntityDespawnPacket(pl.GetEntityId()))
						}
						world.RemovePlayer(pl)
						tracker.ResetEntity(pl.GetEntityId())
					}
				})
			}
			close(done)
			connection.Close()
			return
		}
	}
}

type Server struct {
	World   *level.World
	Tracker *entities.EntityTracker
}


func (s *Server) Run() {
	go s.World.RunCommands()

	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			s.World.Enqueue(func() {
				// For fast time, set it to TickSpeed to 20
				nextTick := (s.World.Tick + s.World.TickSpeed) % 24000
				s.World.AdvanceTick(nextTick, s.Tracker)
				if s.World.Tick%120 == 0 {
					tick := s.World.Tick
					if removed := s.World.PopUnusedChunks(0); len(removed) > 0 {
						ents := s.World.CaptureEntities(removed, 0, s.Tracker.ResetEntity)
						go func() {
							if err := level.SaveChunks(s.World, s.World.WorldDir, removed, ents, 0, tick); err != nil {
								log.Println("Failed to save the s.World:", err)
							}
						}()
					}
					if removed := s.World.PopUnusedChunks(-1); len(removed) > 0 {
						ents := s.World.CaptureEntities(removed, -1, s.Tracker.ResetEntity)
						go func() {
							if err := level.SaveChunks(s.World, s.World.WorldDir, removed, ents, -1, tick); err != nil {
								log.Println("Failed to save the s.World:", err)
							}
						}()
					}
				}
				s.Tracker.Manage(s.World)
				s.World.FlushBlockQueue()
			})
		}
	}()
}
