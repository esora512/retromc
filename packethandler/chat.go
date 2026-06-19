package packethandler

import (
	"strings"

	"fmt"

	entPack "github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func sendDebugMessage(pl *player.Player, lines ...string) {
	for _, line := range lines {
		p := packets.ChatMessagePacket{Message: "\u00A7e" + line}
		pl.Connection.Write(p.Serialize())
	}
}

func handleChatMessageInPacket(p packets.ChatMessagePacket, pl *player.Player, world *level.World) bool {
	message := p.Message
	if strings.HasPrefix(message, "/") {
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
			var lines []string
			if err := world.SaveChanges("saves/world.dat"); err != nil {
				lines = append(lines, fmt.Sprintf("Failed to save world: %v", err))
			} else {
				lines = append(lines, "World saved successfully.")
			}
			if err := player.SaveInventory(pl.Username, pl.Inventory); err != nil {
				lines = append(lines, fmt.Sprintf("Failed to save player data: %v", err))
			} else {
				lines = append(lines, "Player data saved successfully.")
			}
			if err := inventory.SaveContainers("saves/containers.dat"); err != nil {
				lines = append(lines, fmt.Sprintf("Failed to save containers: %v", err))
			} else {
				lines = append(lines, "Containers saved successfully.")
			}
			sendDebugMessage(pl, lines...)
		}

		if strings.HasPrefix(message, "/destroy") {
			var x, y, z int32
			_, err := fmt.Sscanf(message, "/destroy %d %d %d", &x, &y, &z)
			if err != nil {
				sendDebugMessage(pl, fmt.Sprintf("Invalid destroy command format: %s", message))
				return false
			}
			air := level.NewAirBlock()
			world.SetBlock(x, byte(y), z, air)
			blockChange := packets.BlockChangeOutPacket{
				X:         x,
				Y:         byte(y),
				Z:         z,
				BlockType: air.TypeId,
				BlockMeta: air.Metadata,
			}
			world.BroadcastPacket(blockChange.Serialize())
			sendDebugMessage(pl, fmt.Sprintf("Destroyed block at x=%d, y=%d, z=%d", x, y, z))
		}

		if strings.HasPrefix(message, "/gamemode") {
			var mode int
			_, err := fmt.Sscanf(message, "/gamemode %d", &mode)
			if err != nil {
				sendDebugMessage(pl, fmt.Sprintf("Invalid gamemode command format: %s", message))
				return false
			}
			switch mode {
			case 0:
				pl.IsCreative = false
				p.Message = fmt.Sprintf("%s switched to Survival mode", pl.Username)
				world.BroadcastPacket(p.Serialize())
			case 1:
				pl.IsCreative = true
				p.Message = fmt.Sprintf("%s switched to Creative mode", pl.Username)
				world.BroadcastPacket(p.Serialize())
			default:
				p.Message = "\u00A7c" + fmt.Sprintf("Unknown gamemode: %d", mode)
				pl.Connection.Write(p.Serialize())
			}
		}

		if strings.HasPrefix(message, "/kill") {
			command := strings.TrimPrefix(message, "/kill ")
			switch command {
			case "entities":
				entities := world.SnapshotEntities()
				for _, e := range entities {
					if !e.IsPlayer() {
						e.SetHP(0)
					}
				}
			default:
				p.Message = "\u00A7c" + fmt.Sprintf("Server killed %s", pl.Username)
				world.BroadcastPacket(p.Serialize())
				sendSetHealth(pl.Connection, 0)
				pl.SetHP(0)
			}
		}

		if strings.HasPrefix(message, "/time") {
			var subcommand string
			var value int64
			_, err := fmt.Sscanf(message, "/time %s %d", &subcommand, &value)
			if err != nil {
				sendDebugMessage(pl, "Usage: /time set <value>")
				return false
			}
			switch subcommand {
			case "set":
				world.Tick = value
				sendDebugMessage(pl, fmt.Sprintf("Time set to %d", value))
			case "tickspeed":
				world.TickSpeed = value
				sendDebugMessage(pl, fmt.Sprintf("Tickspeed set to %d", value))
			default:
				sendDebugMessage(pl, fmt.Sprintf("Unknown time command: %s", subcommand))
			}
		}

		if strings.HasPrefix(message, "/debug") {
			command := strings.TrimPrefix(message, "/debug ")
			switch command {
			case "water":
				lines := []string{fmt.Sprintf("Water sources in world: %d", len(world.WaterSources))}
				for key := range world.WaterSources {
					lines = append(lines, fmt.Sprintf("  source at x=%d, y=%d, z=%d", key.X, key.Y, key.Z))
				}
				for key := range world.FlowingWater {
					lines = append(lines, fmt.Sprintf("  flowing at x=%d, y=%d, z=%d", key.X, key.Y, key.Z))
				}
				sendDebugMessage(pl, lines...)
			case "fallables":
				lines := []string{fmt.Sprintf("Falling blocks in world: %d", len(world.Fallables))}
				for key := range world.Fallables {
					lines = append(lines, fmt.Sprintf("  at x=%d, y=%d, z=%d", key.X, key.Y, key.Z))
				}
				sendDebugMessage(pl, lines...)
			case "entities":
				entities := world.SnapshotEntities()
				lines := []string{fmt.Sprintf("Entities in world: %d", len(entities))}
				for _, e := range entities {
					x, y, z := e.GetPosition()
					lines = append(lines, fmt.Sprintf("  [%d] at x=%.2f, y=%.2f, z=%.2f", e.GetEntityId(), x, y, z))
					if boat, ok := e.(*entPack.RideableEntity); ok {
						lines = append(lines, fmt.Sprintf("    rideable, passenger: %d", boat.PassengerEntityId))
					}
					if falling, ok := e.(*entPack.BlockEntity); ok {
						lines = append(lines, fmt.Sprintf("    falling block, type=%d, meta=%d, landed=%t", falling.TypeId, falling.Metadata, falling.Landed))
					}
				}
				sendDebugMessage(pl, lines...)
			case "growables":
				lines := []string{fmt.Sprintf("Growables in world: %d", len(world.Growables))}
				for key, e := range world.Growables {
					if crops, ok := e.(*level.Wheat); ok {
						lines = append(lines, fmt.Sprintf("  Type=Wheat, State=%d, StartTick=%d at x=%d, y=%d, z=%d", crops.State, crops.StartTick, key.X, key.Y, key.Z))
					}
					if cactus, ok := e.(*level.Cactus); ok {
						lines = append(lines, fmt.Sprintf("  Type=Cactus, StartTick=%d at x=%d, y=%d, z=%d", cactus.StartTick, key.X, key.Y, key.Z))
					}
					if sapling, ok := e.(*level.Sapling); ok {
						lines = append(lines, fmt.Sprintf("  Type=Sapling, WoodType=%d, StartTick=%d at x=%d, y=%d, z=%d", sapling.WoodType, sapling.StartTick, key.X, key.Y, key.Z))
					}
					if sugarcane, ok := e.(*level.Sugarcane); ok {
						lines = append(lines, fmt.Sprintf("  Type=Sugarcane, StartTick=%d at x=%d, y=%d, z=%d", sugarcane.StartTick, key.X, key.Y, key.Z))
					}
					if dirt, ok := e.(*level.GrowableDirt); ok {
						lines = append(lines, fmt.Sprintf("  Type=GrowableDirt, StartTick=%d at x=%d, y=%d, z=%d", dirt.StartTick, key.X, key.Y, key.Z))
					}
				}
				sendDebugMessage(pl, lines...)
			case "time":
				lines := []string{fmt.Sprintf("World Tick = %d", world.Tick)}
				sendDebugMessage(pl, lines...)
			default:
				sendDebugMessage(pl, "Unknown debug target")
			}
		}

		return true
	}
	p.Message = "<" + pl.Username + "> " + message
	world.BroadcastPacket(p.Serialize())
	return false
}
