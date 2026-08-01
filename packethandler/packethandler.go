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

func HandlePacket(connection net.Conn, reader *bufio.Reader, world *level.World, pl *player.Player, tracker *level.EntityTracker) error {
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
		handleLoginRequestInPacket(connection, packet, world, pl, tracker)
	case packet.PlayerPositionAndLook:
		p := packets.ReadPlayerPositionAndLookInPacket(packetReader)
		handlePlayerPositionAndLookInPacket(connection, p, pl, world)
	case packet.PlayerPosition:
		p := packets.ReadPlayerPositionInPacket(packetReader)
		handlePlayerPositionInPacket(connection, p, pl, world)
	case packet.PlayerOnGround:
		packets.ReadPlayerOnGroundInPacket(packetReader)
	case packet.PlayerLook:
		p := packets.ReadPlayerLookInPacket(packetReader)
		handlePlayerLookInPacket(p, pl, world)
	case packet.EntityAction:
		p := packets.ReadEntityActionInPacket(packetReader)
		handleEntityActionInPacket(p, pl, world)
	case packet.PlayerAnimation:
		p := packets.ReadPlayerAnimationInPacket(packetReader)
		if p.Animation == 1 {
			world.MulticastPacket(packets.ArmSwing(pl), pl)
		}
	case packet.PlayerDigging:
		p := packets.ReadPlayerDiggingInPacket(packetReader)
		handlePlayerDiggingInPacket(connection, p, world, pl)
	case packet.HoldingChange:
		p := packets.ReadHoldingChangeInPacket(packetReader)
		handleHoldingChangeInPacket(p, pl, world)
	case packet.PlayerBlockPlacement:
		p := packets.ReadPlaceInPacket(packetReader)
		handlePlayerBlockPlacementInPacket(connection, p, world, pl, tracker)
	case packet.WindowClick:
		p := packets.ReadWindowClickInPacket(packetReader)
		before := pl.Inventory.PeekItem(pl.HotbarSlot)
		handleWindowClickInPacket(connection, p, world, pl)
		sendCurrentInventory(connection, pl)
		after := pl.Inventory.PeekItem(pl.HotbarSlot)
		if before != after {
			sendEquipmentChangeForHotbarSlot(world, pl)
		}
	case packet.Respawn:
		p := packets.ReadRespawnInPacket(packetReader)
		handleRespawnInPacket(connection, p, world, pl, tracker)
	case packet.CloseWindow:
		p := packets.ReadCloseWindowInPacket(packetReader, pl)
		handleCloseWindowInPacket(p, pl)
	case packet.InteractWithEntity:
		p := packets.ReadInteractWithEntityInPacket(packetReader)
		handleInteractWithEntityInPacket(p, pl, world, tracker)
	case packet.Disconnect:
		p := packets.ReadDisconnectInPacket(packetReader)
		handleDisconnectInPacket(connection, p, world, pl)
	case packet.ChatMessage:
		p := packets.ReadChatMessageInPacket(packetReader)
		isCommand := handleChatMessageInPacket(p, pl, world)
		if isCommand {
			sendCurrentInventory(connection, pl)
		}
	case packet.UpdateSign:
		p := packets.ReadUpdateSignPacket(packetReader)
		handleSignUpdateInPacket(p, world, pl)
	case packet.PlayerInput:
		log.Println("Received PlayerInput packet")
		p := packets.ReadPlayerInputInPacket(packetReader)
		handlePlayerInputInPacket(p, pl, world)
	default:
		log.Printf("Unhandled packet, packet id: 0x%02X", packetId)
	}
	return nil
}
