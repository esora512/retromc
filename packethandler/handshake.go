package packethandler

import (
	"log"
	"net"

	"github.com/leNicDev/retromc/packet/packets"
)

func handleHandshakeInPacket(connection net.Conn, p packets.HandshakeInPacket) {
	log.Printf("Handshake: %+v", p)

	// create handshake out packet
	handshakeOutPacket := packets.HandshakeOutPacket{
		ConnectionHash: "-",
	}
	outData := handshakeOutPacket.Serialize()

	// write handshake out packet
	connection.Write(outData)
}

func handleDisconnectInPacket(connection net.Conn, p packets.DisconnectInPacket) {
	log.Printf("Disconnect: %s", p.Reason)
}
