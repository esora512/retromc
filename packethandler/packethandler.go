package packethandler

import (
	"log"
	"net"

	"bufio"

	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func NewFillContainerPacket(connection net.Conn, pl *player.Player) {
	windowItemsPacket := packets.FillContainerPacket{
		WindowId: 0, // 0 = player inventory
		Count:    int16(pl.Inventory.Size),
		Payload:  pl.Inventory,
	}
	connection.Write(windowItemsPacket.Serialize())
}

func HandlePacket(connection net.Conn, reader *bufio.Reader, world *level.World, pl *player.Player, tracker *entities.EntityTracker) error {
	packetId, err := reader.ReadByte()
	if err != nil {
		log.Println("Failed to read packet id:", err.Error())
		return err
	}
	packetReader := packet.NewReader(reader, packetId)

	switch packetId {
	case packet.KeepAlive:
		packet := packets.ReadKeepAlivePacket(packetReader)
		handleKeepAlivePacket(connection, packet)
	case packet.PreLogin:
		packet := packets.ReadPreLoginPacket(packetReader)
		handlePreLoginPacket(connection, packet)
	case packet.Login:
		packet := packets.ReadLoginPacket(packetReader)
		handleLoginRequestInPacket(connection, packet, world, pl, tracker)
	case packet.PlayerPositionAndRotation:
		p := packets.ReadPlayerPositionAndRotationPacket(packetReader)
		handlePlayerPositionAndRotationPacket(connection, p, pl, world)
	case packet.PlayerPosition:
		p := packets.ReadPlayerPositionPacket(packetReader)
		handlePlayerPositionPacket(connection, p, pl, world)
	case packet.PlayerMovement:
		p := packets.ReadPlayerMovementPacket(packetReader)
		pl.OnGround = p.OnGround
		// TODO: Unhandled, should broadcast player movement to other players
	case packet.PlayerRotation:
		p := packets.ReadPlayerRotationPacket(packetReader)
		handlePlayerRotationPacket(p, pl, world)
	case packet.PlayerAction:
		p := packets.ReadPlayerActionPacket(packetReader)
		handlePlayerActionPacket(p, pl, world)
	case packet.Animation:
		p := packets.ReadAnimationPacket(packetReader)
		if p.Animation == 1 {
			pl.MovementState.ArmSwing = true 
		}
	case packet.MineBlock:
		p := packets.ReadPlayerMineBlockPacket(packetReader)
		handleMineBlockPacket(connection, p, world, pl)
	case packet.SetHotbarSlot:
		p := packets.ReadSetHotbarSlot(packetReader)
		handleSetHotbarSlot(p, pl, world)
	case packet.PlaceBlock:
		p := packets.ReadPlaceBlockPacket(packetReader)
		handlePlaceBlockPacket(connection, p, world, pl)
	case packet.ClickSlot:
		p := packets.ReadClickSlotPacket(packetReader)
		before := pl.Inventory.PeekItem(pl.HotbarSlot)
		handleClickSlotPacket(connection, p, world, pl)
		NewFillContainerPacket(connection, pl)
		after := pl.Inventory.PeekItem(pl.HotbarSlot)
		if before != after {
			sendEquipmentChangeForHotbarSlot(world, pl)
		}
	case packet.Respawn:
		p := packets.ReadRespawnPacket(packetReader)
		handleRespawnInPacket(connection, p, world, pl)
	case packet.CloseContainer:
		p := packets.ReadCloseContainerPacket(packetReader, pl)
		handleCloseContainerPacket(p, pl)
	case packet.InteractWithEntity:
		p := packets.ReadInteractWithEntityPacket(packetReader)
		log.Printf("%+v", p)
		handleInteractWithEntityPacket(p, pl, world, tracker)
	case packet.Disconnect:
		p := packets.ReadDisconnectPacket(packetReader)
		handleDisconnectPacket(connection, p, world, pl)
	case packet.ChatMessage:
		p := packets.ReadChatMessagePacket(packetReader)
		isCommand := handleChatMessageInPacket(p, pl, world, tracker)
		if isCommand {
			NewFillContainerPacket(connection, pl)
		}
	case packet.UpdateSign:
		p := packets.ReadUpdateSignPacket(packetReader)
		handleUpdateSignPacket(p, world, pl)
	case packet.PlayerInput:
		log.Println("Received PlayerInput packet")
		p := packets.ReadPlayerInputPacket(packetReader)
		handlePlayerInputPacket(p, pl, world)
	default:
		log.Printf("Unhandled packet, packet id: 0x%02X", packetId)
	}
	return nil
}
