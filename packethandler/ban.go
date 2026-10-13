package packethandler

import (
	"net"
	"time"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func remoteIP(conn net.Conn) string {
	host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	return host
}

func kickConnection(conn net.Conn, reason string) {
	kick := packets.DisconnectPacket{Reason: reason}
	conn.Write(kick.Serialize())
	time.AfterFunc(time.Second, func() { conn.Close() })
}

func handleBanCommand(pl *player.Player, world *level.World, args []string) {
	if len(args) != 1 {
		sendUsage(pl, "/ban")
		return
	}
	ip := ""
	if target := world.GetFirstPlayerByName(args[0]); target != nil {
		ip = remoteIP(target.Connection)
		kickConnection(target.Connection, "You are banned from this server")
	}
	world.Ban(args[0], ip)
	sendDebugMessage(pl, "Banned "+args[0])
}
