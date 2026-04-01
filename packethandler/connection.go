package packethandler

import (
	"log"
	"net"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func handleHandshakeInPacket(connection net.Conn, p packets.HandshakeInPacket) {
	handshakeOutPacket := packets.HandshakeOutPacket{
		ConnectionHash: "-",
	}
	outData := handshakeOutPacket.Serialize()
	connection.Write(outData)
}

func handleDisconnectInPacket(connection net.Conn, p packets.DisconnectInPacket) {
	log.Printf("Disconnect: %s", p.Reason)
}

func handleKeepAliveInPacket(connection net.Conn, p packets.KeepAliveInPacket) {
	//log.Printf("KeepAlive: %+v", p)
	// create keep alive out packet
	keepAliveOutPacket := packets.KeepAliveOutPacket{}
	outData := keepAliveOutPacket.Serialize()

	// write keep alive out packet
	_, err := connection.Write(outData)
	if err != nil {
		log.Println("Failed to write keep alive out packet:", err.Error())
	}
}

func handleLoginRequestInPacket(connection net.Conn, p packets.LoginRequestInPacket, world *level.World, pl *player.Player) {
	pl.Username = p.Username
	sendLoginResponse(connection, pl)
	sendChunks(connection, world)
	sendInventory(connection, pl)
	sendPlayerPositionAndLook(connection)
	spawnPacket := packets.SpawnPlayerEntityPacket(pl)
	// Inform other players of the new player
	world.MulticastPacket(spawnPacket, pl)
	world.ForEachPlayer(func(other *player.Player) {
		if other == pl {
			return
		}
		packets.SetEquipment(pl, func(b []byte) {
			other.Connection.Write(b)
		})
		chatPacket := packets.ChatMessagePacket{
			Message: "\u00a7e" + pl.Username + " joined the game",
		}
		other.Connection.Write(chatPacket.Serialize())
	})

	// Inform the new player of other players
	world.ForEachPlayer(func(other *player.Player) {
		if other == pl {
			return
		}
		spawnPacket := packets.SpawnPlayerEntityPacket(other)
		pl.Connection.Write(spawnPacket)
		packets.SetEquipment(other, func(b []byte) {
			pl.Connection.Write(b)
		})
	})
}

func sendLoginResponse(connection net.Conn, pl *player.Player) {
	outPacket := packets.LoginResponseOutPacket{
		EntityId:  pl.EntityId,
		MapSeed:   0,
		Dimension: 0,
	}
	outData := outPacket.Serialize()

	connection.Write(outData)
	pl.LoggedIn = true
}

