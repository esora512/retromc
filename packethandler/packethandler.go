package packethandler

import (
	"log"
	"net"

	"bufio"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func sendCurrentInventory(connection net.Conn, pl *player.Player) {
	windowItemsPacket := packets.WindowItemsOutPacket{
		WindowId: 0, // 0 = player inventory
		Count:    int16(pl.Inventory.Size),
		Payload:  pl.Inventory,
	}
	connection.Write(windowItemsPacket.Serialize())
}

func HandlePacket(connection net.Conn, reader *bufio.Reader, world *level.World, pl *player.Player) error {
	// read packet
	packetId, err := reader.ReadByte()
	if err != nil {
		log.Println("Failed to read packet id:", err.Error())
		return err
	}
	packetReader := packet.NewReader(reader, packetId)

	switch packetId {
	case packet.KeepAlive:
		packet := packets.ReadKeepAliveInPacket(packetReader)
		handleKeepAliveInPacket(connection, packet)
	case packet.Handshake:
		packet := packets.ReadHandshakeInPacket(packetReader)
		handleHandshakeInPacket(connection, packet)
	case packet.LoginRequest:
		packet := packets.ReadLoginRequestInPacket(packetReader)
		handleLoginRequestInPacket(connection, packet, world, pl)
	case packet.PlayerPositionAndLook:
		p := packets.ReadPlayerPositionAndLookInPacket(packetReader)
		handlePlayerPositionAndLookInPacket(connection, p, pl)
	case packet.PlayerPosition:
		p := packets.ReadPlayerPositionInPacket(packetReader)
		handlePlayerPositionInPacket(connection, p, pl)
	case packet.PlayerOnGround:
		packets.ReadPlayerOnGroundInPacket(packetReader)
	case packet.PlayerLook:
		packets.ReadPlayerLookInPacket(packetReader)
	case packet.EntityAction:
		packets.ReadEntityActionInPacket(packetReader)
	case packet.PlayerAnimation:
		packets.ReadPlayerAnimationInPacket(packetReader)
	case packet.PlayerDigging:
		p := packets.ReadPlayerDiggingInPacket(packetReader)
		handlePlayerDiggingInPacket(connection, p, world, pl)
	case packet.HoldingChange:
		p := packets.ReadHoldingChangeInPacket(packetReader)
		handleHoldingChangeInPacket(p, pl)
	case packet.PlayerBlockPlacement:
		p := packets.ReadPlaceInPacket(packetReader)
		handlePlayerBlockPlacementInPacket(connection, p, world, pl)
	case packet.WindowClick:
		p := packets.ReadWindowClickInPacket(packetReader)
		//log.Printf("Buffer size before: %d", reader.Buffered())
		handleWindowClickInPacket(connection, p, pl)
		sendCurrentInventory(connection, pl)
	case packet.Respawn:
		p := packets.ReadRespawnInPacket(packetReader)
		handleRespawnInPacket(connection, p, world, pl)
	case packet.CloseWindow:
		p := packets.ReadCloseWindowInPacket(packetReader)
		handleCloseWindowInPacket(p)
	default:
		log.Printf("Unhandled packet, packet id: %04x", packetId)
	}
	return nil
}
