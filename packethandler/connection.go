package packethandler

import (
	"fmt"
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

func handleDisconnectInPacket(connection net.Conn, p packets.DisconnectInPacket, world *level.World, pl *player.Player) {
	log.Printf("%s", p.Reason)
	chatPacket := packets.ChatMessagePacket{
		Message: "\u00a7e" + pl.Username + " left the game",
	}
	world.BroadcastPacket(chatPacket.Serialize())
	world.BroadcastPacket(packets.EntityDespawnPacket(pl.GetEntityId()))
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

func handleLoginRequestInPacket(connection net.Conn, p packets.LoginRequestInPacket, world *level.World, pl *player.Player, tracker *level.EntityTracker) {
	pl.Username = p.Username
	unlock := world.LockSession(p.Username)
	defer unlock()

	if old, ok := world.GetPlayerByUsername(pl.Username); ok && old != pl {
		world.BroadcastPacket(packets.PlayerEntityDespawnPacket(old))
		world.RemovePlayer(old)
		tracker.Remove(old.GetEntityId())
		world.UnloadPlayerChunks(old)
		old.Connection.Close()
	}

	data, err := level.LoadPlayerData(world.WorldDir, pl.Username)
	if err != nil {
		log.Printf("Failed to load player inventory for %s : %v", pl.Username, err)
	}
	level.ApplyPlayerData(pl, data)
	world.AddPlayer(pl)

	sendLoginResponse(connection, world, pl)
	updateChunks(world, pl.X, pl.Z, pl)
	sendInventory(connection, pl, world)
	sendPlayerPositionAndLook(connection, pl.X, pl.Z)

	serverPacket1 := packets.ChatMessagePacket{
		Message: "\u00a7e" + fmt.Sprintf("Server runs on retromc (dev/%s)", world.CommitHash),
	}
	pl.Connection.Write(serverPacket1.Serialize())

	chatPacket := packets.ChatMessagePacket{
		Message: "\u00a7e" + pl.Username + " joined the game",
	}
	world.BroadcastPacket(chatPacket.Serialize())
	pl.LoggedIn = true
	//log.Printf("Login %s at x=%f, y=%f, z=%f", pl.Username, pl.X, pl.Y, pl.Z)
}

func sendLoginResponse(connection net.Conn, w *level.World, pl *player.Player) {
	outPacket := packets.LoginResponseOutPacket{
		EntityId:  pl.EntityId,
		MapSeed:   w.Seed,
		Dimension: 0,
	}
	outData := outPacket.Serialize()
	connection.Write(outData)
}
