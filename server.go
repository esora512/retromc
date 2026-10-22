package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"net"
	"os"
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

func main() {
	host := flag.String("host", "localhost", "Address to bind the server to")
	port := flag.String("port", "25565", "Port to bind the server to")

	wType := flag.String("wt", "Default", "World type for world generation")


	logFile := flag.String("log", "server.log", "File to append the server log to")
	offline := flag.Bool("offline", false, "Skip username verification through the Betacraft auth proxy")

	flag.Parse()
	if f, err := os.OpenFile(*logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		log.SetOutput(io.MultiWriter(os.Stderr, f))
	} else {
		log.Println("Failed to open log file:", err)
	}
	world := level.NewWorld(GitCommit, 3257840388504953787, level.GetWorldType(*wType))
	b2 := setupB2(world)
	if err := world.LoadAccounts(); err != nil {
		log.Fatalln("Failed to load players.json:", err)
	}
	if err := world.LoadBans(); err != nil {
		log.Fatalln("Failed to load bans:", err)
	}
	world.OnlineMode = !*offline
	if !world.OnlineMode {
		log.Println("WARNING: online mode is off, anyone can join with any username")
	}

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
	world.SetSendLightRegion(packets.BroadcastChunkRegion)
	world.SetBroadcastTime(packethandler.BroadcastTime)
	world.SetBroadcastWorldMsg(packethandler.BroadcastWorldMsg)
	world.SetSendSetHealth(packethandler.SendSetHealth)
	world.SetHurtPlayer(packethandler.HurtPlayer)
	world.SetAttackEntity(packethandler.AttackEntity)
	world.SetDropItemFromMinedBlock(packethandler.DropItemFromMinedBlock)
	world.SetCloseContainer(packethandler.ForceCloseContainer)

	world.SetSpawnPlayer(packets.NewSpawnPlayerPacket)
	world.SetSpawnObject(packets.NewSpawnObjectPacket)
	world.SetSpawnMob(packets.SpawnMob)
	world.SetSpawnItem(packets.NewSpawnItem)
	world.SetSendEquipment(packets.SetEquipment3)
	world.SetDespawnEntity(packets.NewEntityDespawnPacket)

	world.SetNewAnimationPacket(packethandler.NewAnimationPacket)
	world.SetNewEntityMetadataPacket(packets.NewEntityMetadataPacket)

	world.PrepareNetherSpawn()

	entityTracker := entities.NewEntityTracker()
	server := Server{World: world, Tracker: entityTracker}
	startDiscord(world, *host, *port)
	startBetacraftHeartbeat(world)
	runOnRender(&server, b2)
	server.Run()
	startPersistence(world, b2)

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
							world.RelayPresence(pl.Username, pl.Username+" left the game", world.OnlineCount()-1)
							world.BroadcastPacket(packets.NewEntityDespawnPacket(pl.GetEntityId()))
						}
						packethandler.ClearSkywalk(world, pl)
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
		var ticks int64
		for range ticker.C {
			s.World.Enqueue(func() {
				ticks++
				if s.World.Frozen {
					s.World.BroadcastTime(s.World.TimeTick)
					s.World.TickPlayers()
				} else {
					for range s.World.Speed {
						s.World.AdvanceTick(s.World.Tick+1, s.Tracker)
					}
				}
				if ticks%120 == 0 {
					tick := s.World.Tick
					for _, dim := range level.Dims {
						removed := s.World.PopUnusedChunks(dim)
						if len(removed) == 0 {
							continue
						}
						ents := s.World.CaptureEntities(removed, dim, s.Tracker.ResetEntity)
						save := s.World.PrepareSave(removed, ents, dim)
						go func() {
							if err := level.SaveChunks(s.World, save, tick); err != nil {
								log.Println("Failed to save the s.World:", err)
							}
						}()
					}
				}
				s.Tracker.Manage(s.World)
				s.World.FlushBlockQueue()
				s.World.FlushLight()
				packethandler.FlushPendingChunks(s.World)
			})
		}
	}()
}
