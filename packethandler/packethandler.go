package packethandler

import (
	"log"
	"net"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/packet/packets"
)

func HandlePacket(connection net.Conn, data *[]byte, world *level.World) {
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
		handleLoginRequestInPacket(connection, packet, world)
	case packet.PlayerPositionAndLook:
		packets.ReadPlayerPositionAndLookInPacket(data)
		//handlePlayerPositionAndLookInPacket(connection, packet)
	case packet.PlayerPosition:
		packets.ReadPlayerPositionInPacket(data)
		//handlePlayerPositionInPacket(connection, packet)
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
		handlePlayerDiggingInPacket(connection, p, world)
	case packet.HoldingChange:
		packets.ReadHoldingChangeInPacket(data)
	case packet.PlayerBlockPlacement:
		p := packets.ReadPlaceInPacket(data)
		handlePlayerBlockPlacementInPacket(connection, p, world)
	default:
		log.Printf("Unhandled packet, packet id: %04x", p.PacketId)
	}
}
