package packethandler

import (
	"log"
	"net"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func HandlePacket(connection net.Conn, data *[]byte, world *level.World, pl *player.Player) {
	// read packet
	p := packet.ReadPacket(data)
	switch p.PacketId {
	case packet.KeepAlive:
		packet := packets.ReadKeepAliveInPacket(data)
		handleKeepAliveInPacket(connection, packet)
	case packet.Handshake:
		packet := packets.ReadHandshakeInPacket(data)
		handleHandshakeInPacket(connection, packet)
	case packet.LoginRequest:
		packet := packets.ReadLoginRequestInPacket(data)
		handleLoginRequestInPacket(connection, packet, world, pl)
	case packet.PlayerPositionAndLook:
		packets.ReadPlayerPositionAndLookInPacket(data)
	case packet.PlayerPosition:
		packets.ReadPlayerPositionInPacket(data)
	case packet.PlayerOnGround:
		packets.ReadPlayerOnGroundInPacket(data)
	case packet.PlayerLook:
		packets.ReadPlayerLookInPacket(data)
	case packet.EntityAction:
		packets.ReadEntityActionInPacket(data)
	case packet.PlayerAnimation:
		packets.ReadPlayerAnimationInPacket(data)
	case packet.PlayerDigging:
		p := packets.ReadPlayerDiggingInPacket(data)
		handlePlayerDiggingInPacket(connection, p, world, pl)
	case packet.HoldingChange:
		p := packets.ReadHoldingChangeInPacket(data)
		handleHoldingChangeInPacket(p, pl)
	case packet.PlayerBlockPlacement:
		p := packets.ReadPlaceInPacket(data)
		handlePlayerBlockPlacementInPacket(connection, p, world, pl)
	case packet.WindowClick:
		p := packets.ReadWindowClickInPacket(data)
		handleWindowClickInPacket(p, pl)
	default:
		log.Printf("Unhandled packet, packet id: %04x", p.PacketId)
	}
}
