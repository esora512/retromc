package packethandler

import (
	"fmt"
	"log"
	"net"

	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func handlePreLoginPacket(connection net.Conn, p packets.PreLoginPacket) {
	pkt := packets.PreLoginPacket{
		ConnectionHash: "-",
	}
	outData := pkt.Serialize()
	connection.Write(outData)
}

func NewLeftGameMsg(username string) []byte {
	chatPacket := packets.ChatMessagePacket{
		Message: "\u00a7e" + username + " left the game",
	}
	return chatPacket.Serialize()
}

func handleDisconnectPacket(p packets.DisconnectPacket, world *level.World, pl *player.Player) {
	log.Printf("%s left the game (%s)", pl.Username, p.Reason)
	chatPacket := packets.ChatMessagePacket{
		Message: "\u00a7e" + pl.Username + " left the game",
	}

	world.SavePlayer(pl)
	world.BroadcastPacket(chatPacket.Serialize())
	world.BroadcastPacket(packets.NewEntityDespawnPacket(pl.GetEntityId()))
	pl.LoggedIn = false
}

func handleKeepAlivePacket(connection net.Conn, p packets.KeepAlivePacket) {
	pkt := packets.KeepAlivePacket{}
	outData := pkt.Serialize()
	// write keep alive out packet
	_, err := connection.Write(outData)
	if err != nil {
		log.Println("Failed to write keep alive out packet:", err.Error())
	}
}

func handleLoginRequestInPacket(connection net.Conn, p packets.LoginPacket, world *level.World, pl *player.Player, tracker *entities.EntityTracker) {
	if n := len([]rune(p.Username)); n == 0 || n > 16 {
		kick := packets.DisconnectPacket{Reason: "Invalid username"}
		connection.Write(kick.Serialize())
		connection.Close()
		return
	}
	pl.Username = p.Username

	if old, ok := world.GetPlayerByUsername(pl.Username); ok && old != pl {
		world.BroadcastPacket(packets.NewEntityDespawnPacket(old.GetEntityId()))
		world.RemovePlayer(old)
		tracker.ResetEntity(old.GetEntityId())
		old.Connection.Close()
	}

	data, err := level.LoadPlayerData(world.WorldDir, pl.Username)
	if err != nil {
		log.Printf("Failed to load player inventory for %s : %v", pl.Username, err)
	}
	level.ApplyPlayerData(pl, data)
	world.AddPlayer(pl)

	sendLoginResponse(connection, world, pl)

	if pl.Y <= -1000000 {
		pl.Y = 80
	}

	initialUpdateChunks(world, pl.X, pl.Z, pl, func() {
		finishLogin(connection, world, pl)
	})
}

func finishLogin(connection net.Conn, world *level.World, pl *player.Player) {
	sendInventory(connection, pl, world)
	sendPlayerPositionAndLook(connection, pl.X, pl.Z, pl.Y)

	serverPacket1 := packets.ChatMessagePacket{
		Message: "\u00a7e" + fmt.Sprintf("Server runs on retromc (dev/%s)", world.CommitHash),
	}
	pl.Connection.Write(serverPacket1.Serialize())

	chatPacket := packets.ChatMessagePacket{
		Message: "\u00a7e" + pl.Username + " joined the game",
	}
	world.BroadcastPacket(chatPacket.Serialize())
	pl.LoggedIn = true
	// TODO: Figure out a better way to op players
	pl.IsOp = true
	SendSetHealth(connection, uint16(pl.HP))

	// if world.OppedUsernames[pl.Username] {
	// 	pl.IsOp = true
	// }
}

func sendLoginResponse(connection net.Conn, w *level.World, pl *player.Player) {
	outPacket := packets.LoginPacket{
		EntityId:  pl.EntityId,
		MapSeed:   w.Seed,
		Dimension: byte(pl.Dimension),
	}
	outData := outPacket.Serialize()
	connection.Write(outData)
}
