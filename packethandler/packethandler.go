package packethandler

import (
	"fmt"
	"log"
	"net"
	"runtime/debug"

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


func HandlePacket(connection net.Conn, reader *bufio.Reader, world *level.World, pl *player.Player, tracker *entities.EntityTracker) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered from panic decoding packet from %s: %v\n%s", connection.RemoteAddr(), r, debug.Stack())
			err = fmt.Errorf("panic decoding packet: %v", r)
		}
	}()

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
		world.Enqueue(func() {
			handleLoginRequestInPacket(connection, packet, world, pl, tracker)
		})
	case packet.PlayerPositionAndRotation:
		p := packets.ReadPlayerPositionAndRotationPacket(packetReader)
		world.Enqueue(func() {
			handlePlayerPositionAndRotationPacket(connection, p, pl, world)
		})
	case packet.PlayerPosition:
		p := packets.ReadPlayerPositionPacket(packetReader)
		world.Enqueue(func() {
			handlePlayerPositionPacket(connection, p, pl, world)
		})
	case packet.PlayerMovement:
		p := packets.ReadPlayerMovementPacket(packetReader)
		world.Enqueue(func() {
			pl.OnGround = p.OnGround
			// TODO: Unhandled, should broadcast player movement to other players
		})
	case packet.PlayerRotation:
		p := packets.ReadPlayerRotationPacket(packetReader)
		world.Enqueue(func() {
			handlePlayerRotationPacket(p, pl, world)
		})
	case packet.PlayerAction:
		p := packets.ReadPlayerActionPacket(packetReader)
		world.Enqueue(func() {
			handlePlayerActionPacket(p, pl, world)
		})
	case packet.Animation:
		p := packets.ReadAnimationPacket(packetReader)
		if p.Animation == 1 {
			world.Enqueue(func() {
				pl.MovementState.ArmSwing = true
			})
		}
	case packet.MineBlock:
		p := packets.ReadPlayerMineBlockPacket(packetReader)
		world.Enqueue(func() {
			handleMineBlockPacket(connection, p, world, pl)
		})
	case packet.SetHotbarSlot:
		p := packets.ReadSetHotbarSlot(packetReader)
		world.Enqueue(func() {
			handleSetHotbarSlot(p, pl, world)
		})
	case packet.PlaceBlock:
		p := packets.ReadPlaceBlockPacket(packetReader)
		world.Enqueue(func() {
			handlePlaceBlockPacket(connection, p, world, pl)
		})
	case packet.ClickSlot:
		p := packets.ReadClickSlotPacket(packetReader)
		world.Enqueue(func() {
			before := pl.Inventory.PeekItem(pl.HotbarSlot)
			handleClickSlotPacket(connection, p, world, pl)
			NewFillContainerPacket(connection, pl)
			after := pl.Inventory.PeekItem(pl.HotbarSlot)
			if before != after {
				sendEquipmentChangeForHotbarSlot(world, pl)
			}
		})
	case packet.Respawn:
		p := packets.ReadRespawnPacket(packetReader)
		world.Enqueue(func() {
			handleRespawnInPacket(connection, p, world, pl)
		})
	case packet.CloseContainer:
		p := packets.ReadCloseContainerPacket(packetReader, pl)
		world.Enqueue(func() {
			handleCloseContainerPacket(p, pl)
		})
	case packet.InteractWithEntity:
		p := packets.ReadInteractWithEntityPacket(packetReader)
		world.Enqueue(func() {
			handleInteractWithEntityPacket(p, pl, world, tracker)
		})
	case packet.Disconnect:
		p := packets.ReadDisconnectPacket(packetReader)
		world.Enqueue(func() {
			handleDisconnectPacket(p, world, pl)
		})
	case packet.ChatMessage:
		p := packets.ReadChatMessagePacket(packetReader)
		world.Enqueue(func() {
			isCommand := handleChatMessageInPacket(p, pl, world, tracker)
			if isCommand {
				NewFillContainerPacket(connection, pl)
			}
		})
	case packet.UpdateSign:
		p := packets.ReadUpdateSignPacket(packetReader)
		world.Enqueue(func() {
			handleUpdateSignPacket(p, world, pl)
		})
	case packet.PlayerInput:
		log.Println("Received PlayerInput packet")
		p := packets.ReadPlayerInputPacket(packetReader)
		world.Enqueue(func() {
			handlePlayerInputPacket(p, pl, world)
		})
	default:
		log.Printf("Unhandled packet, packet id: 0x%02X", packetId)
	}
	return nil
}
