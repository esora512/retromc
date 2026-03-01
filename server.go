package main

import (
	"log"
	"net"
	"time"

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

func handleConnection(connection net.Conn, world *level.World) {
	pl := player.NewPlayer(connection)
	done := make(chan struct{})

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
			case <-done:
				return
			}
		}
	}()

	var buf []byte
	for {
		buf = make([]byte, 1024)

		_, err := connection.Read(buf)
		if err != nil {
			log.Println("Connection closed:", err.Error())
			close(done)
			connection.Close()
			return
		}

		packethandler.HandlePacket(connection, &buf, world, pl)
	}
}
