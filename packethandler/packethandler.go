package packethandler

import (
	"log"
	"net"

	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/packet/packets"
)

func HandlePacket(connection net.Conn, data *[]byte) {
	// read packet
	p := packet.ReadPacket(data)

	log.Printf("Received packet: %+v", p)

	switch p.PacketId {
	case packet.KeepAlive:
		packet := packets.ReadKeepAliveInPacket(data)
		handleKeepAliveInPacket(connection, packet)
	case packet.Handshake:
		packet := packets.ReadHandshakeInPacket(data)
		handleHandshakeInPacket(connection, packet)
		break
	case packet.LoginRequest:
		packet := packets.ReadLoginRequestInPacket(data)
		handleLoginRequestInPacket(connection, packet)
		break
	case packet.PlayerPositionAndLook:
		packet := packets.ReadPlayerPositionAndLookInPacket(data)
		handlePlayerPositionAndLookInPacket(connection, packet)
		break
	case packet.PlayerPosition:
		packet := packets.ReadPlayerPositionInPacket(data)
		handlePlayerPositionInPacket(connection, packet)
		break
	default:
		log.Printf("Ignoring unknown packet id: %x", p.PacketId)
	}
}
