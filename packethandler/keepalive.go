package packethandler

import (
	"log"
	"net"

	"github.com/leNicDev/retromc/packet/packets"
)

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
