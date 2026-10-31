package packethandler

import (
	"log"
	"net"
	"slices"
	"strings"
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
	sendDebugMessage(pl, BanCommand(world, pl.Username, args)...)
}

// BanCommand runs /ban on behalf of by and returns the reply lines
func BanCommand(world *level.World, by string, args []string) []string {
	if len(args) != 1 {
		return []string{"Usage: " + commandUsage("/ban")}
	}
	name := args[0]
	ips := slices.Clone(world.IPsOf(name))
	for _, p := range world.Players {
		if ip := remoteIP(p.Connection); strings.EqualFold(p.Username, name) && ip != "" && !slices.Contains(ips, ip) {
			ips = append(ips, ip)
		}
	}
	for _, p := range world.Players {
		if strings.EqualFold(p.Username, name) || slices.Contains(ips, remoteIP(p.Connection)) {
			kickConnection(p.Connection, "You are banned from this server")
		}
	}
	world.Ban(name, ips)
	log.Printf("%s banned %s (IPs %v)", by, name, ips)
	if len(ips) == 0 {
		return []string{"Banned " + name + " (no known IP, name only)"}
	}
	return []string{"Banned " + name + " and IPs " + strings.Join(ips, ", ")}
}
