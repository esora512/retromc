package packethandler

import (
	"log"
	"strings"

	"fmt"

	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func handleChatMessageInPacket(p packets.ChatMessagePacket, pl *player.Player, world *level.World) bool {
	message := p.Message
	if strings.HasPrefix(message, "/") {
		//log.Printf("Command: %s", message)
		if strings.HasPrefix(message, "/give") {
			command := strings.TrimPrefix(message, "/give ")
			before := pl.Inventory.PeekItem(pl.HotbarSlot)
			pl.GivePlayer(command)
			after := pl.Inventory.PeekItem(pl.HotbarSlot)
			if before != after {
				sendEquipmentChangeForHotbarSlot(world, pl)
			}
		}
		if strings.HasPrefix(message, "/save") {
			if err := world.SaveChanges("saves/world.dat"); err != nil {
				log.Println("Failed to save world:", err)
			} else {
				log.Println("World saved successfully.")
			}
			if err := player.SaveInventory(pl.Username, pl.Inventory); err != nil {
				log.Println("Failed to save player data:", err)
			} else {
				log.Println("Player data saved successfully.")
			}
			if err := inventory.SaveContainers("saves/containers.dat"); err != nil {
				log.Println("Failed to save containers:", err)
			} else {
				log.Println("Containers saved successfully.")
			}
		}

		if strings.HasPrefix(message, "/destroy") {
			var x, y, z int32
			_, err := fmt.Sscanf(message, "/destroy %d %d %d", &x, &y, &z)
			if err != nil {
				log.Printf("Invalid destroy command format: %s", message)
				return false
			}
			log.Printf("Destroying block x=%d, y=%d, z=%d", x, y, z)
			air := level.NewAirBlock()
			world.SetBlock(x, byte(y), z, air)
			blockChange := packets.BlockChangeOutPacket{
				X:        x,
				Y:         byte(y),
				Z:         z,
				BlockType: air.TypeId,
				BlockMeta: air.Metadata,
			}
			world.BroadcastPacket(blockChange.Serialize())
		}

		if strings.HasPrefix(message, "/gamemode") {
			var mode int
			_, err := fmt.Sscanf(message, "/gamemode %d", &mode)
			if err != nil {
				log.Printf("Invalid gamemode command format: %s", message)
				return false
			}
			switch mode {
			case 0:
				pl.IsCreative = false
				log.Printf("Player %s switched to Survival mode", pl.Username)
			case 1:
				pl.IsCreative = true
				log.Printf("Player %s switched to Creative mode", pl.Username)
			default:
				log.Printf("Unknown gamemode: %d", mode)
			}
		}

		if strings.HasPrefix(message, "/debug") {
			command := strings.TrimPrefix(message, "/debug ")
			switch command {
			case "water":
				log.Printf("Water sources in world: %d", len(world.WaterSources))
				for key := range world.WaterSources {
					log.Printf("Water source at x=%d, y=%d, z=%d", key.X, key.Y, key.Z)
				}
			default:
				log.Printf("Unknown debug command: %s", command)
			}
		}

		return true
	}
	p.Message = "<" + pl.Username + "> " + message
	world.BroadcastPacket(p.Serialize())
	return false
}
