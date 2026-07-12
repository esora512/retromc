package packethandler

import (
	"regexp"
	"sort"
	"strings"

	"fmt"

	"runtime"

	"github.com/leNicDev/retromc/constants"
	entPack "github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

func LogMemStats() (string, string, string, string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return fmt.Sprintf("Alloc = %v MiB", m.Alloc/1024/1024),
		fmt.Sprintf("Sys = %v MiB", m.Sys/1024/1024),
		fmt.Sprintf("TotalAlloc = %v MiB", m.TotalAlloc/1024/1024),
		fmt.Sprintf("NumGC = %v", m.NumGC)
}

// commandHelp maps each command name to a short description and usage string.
// Used by both /help (overview) and /help <command> (detail), and commands
// fall back to their usage entry here when called with bad/missing args.
var commandHelp = []struct {
	Name  string
	Usage string
}{
	{"/give", "/give <item>"},
	{"/save", "/save"},
	{"/destroy", "/destroy <x> <y> <z>"},
	{"/gamemode", "/gamemode <0|1>"},
	{"/kill", "/kill [entities]"},
	{"/time", "/time <set | tickspeed> <value>"},
	{"/debug", "/debug <water | fallables | entities | growables | time | block | furnaces>"},
	{"/help", "/help [command]"},
	{"/tp", "/tp <x> <y> <z> | tp <p1> <p2>"},
	{"/size", "/size"},
}

func sendDebugMessage(pl *player.Player, lines ...string) {
	for _, line := range lines {
		p := packets.ChatMessagePacket{Message: "\u00A7e" + line}
		pl.Connection.Write(p.Serialize())
	}
}

// sendUsage looks up a command by name and prints its usage line.
// If the command isn't found (shouldn't normally happen), it's a no-op.
func sendUsage(pl *player.Player, name string) {
	for _, c := range commandHelp {
		if c.Name == name {
			sendDebugMessage(pl, fmt.Sprintf("Usage: %s", c.Usage))
			return
		}
	}
}

func handleHelpCommand(pl *player.Player, message string) {
	arg := strings.TrimSpace(strings.TrimPrefix(message, "/help"))
	if arg == "" {
		lines := []string{"Available commands:"}
		for _, c := range commandHelp {
			lines = append(lines, c.Name)
		}
		sendDebugMessage(pl, lines...)
		return
	}

	// Allow looking up with or without the leading slash, e.g. "/help give" or "/help /give"
	if !strings.HasPrefix(arg, "/") {
		arg = "/" + arg
	}
	for _, c := range commandHelp {
		if c.Name == arg {
			sendDebugMessage(pl, c.Name, fmt.Sprintf("Usage: %s", c.Usage))
			return
		}
	}
	sendDebugMessage(pl, fmt.Sprintf("Unknown command: %s", arg))
}

func handleChatMessageInPacket(p packets.ChatMessagePacket, pl *player.Player, world *level.World) bool {
	message := p.Message
	if strings.HasPrefix(message, "/") {
		if strings.HasPrefix(message, "/tp") {
			args := strings.Fields(strings.TrimPrefix(message, "/tp"))

			switch len(args) {
			case 3:
				var x, y, z float64
				_, err := fmt.Sscanf(message, "/tp %f %f %f", &x, &y, &z)
				if err != nil {
					sendUsage(pl, "/tp")
					return false
				}
				pl.SetPosition(x, y, z)
				BroadcastTeleportPlayer(world, pl, x, y, z, byte(pl.Yaw))
				return false

			case 2:
				player1Name, player2Name := args[0], args[1]
				pl1 := world.GetFirstPlayerByName(player1Name)
				pl2 := world.GetFirstPlayerByName(player2Name)
				if pl1 == nil || pl2 == nil {
					sendDebugMessage(pl, fmt.Sprintf("One or both players not found: %s, %s", player1Name, player2Name))
					return false
				}
				x, y, z := pl2.GetPosition()
				pl1.SetPosition(x, y, z)
				BroadcastTeleportPlayer(world, pl1, x, y, z, byte(pl1.Yaw))
				return false

			default:
				sendUsage(pl, "/tp")
				return false
			}
		}

		if strings.HasPrefix(message, "/tp") {
			var x, y, z float64
			_, err := fmt.Sscanf(message, "/tp %f %f %f", &x, &y, &z)
			if err != nil {
				sendUsage(pl, "/tp")
				return false
			}
			pl.SetPosition(x, y, z)
			BroadcastTeleport(world, pl, x, y, z, byte(pl.Yaw))
			return false
		}

		if strings.HasPrefix(message, "/size") {
			sendDebugMessage(pl, fmt.Sprintf("World size = %s", world.SizeString()))
			alloc, sys, totalAlloc, numGC := LogMemStats()
			sendDebugMessage(pl, alloc)
			sendDebugMessage(pl, sys)
			sendDebugMessage(pl, totalAlloc)
			sendDebugMessage(pl, numGC)
		}

		if strings.HasPrefix(message, "/give") {
			command := strings.TrimPrefix(message, "/give ")
			if command == "" || command == "/give" {
				sendUsage(pl, "/give")
				return false
			}
			if command == "help" || command == "?" {
				printGiveHelp(pl, "")
				return false
			}
			if strings.HasPrefix(command, "? ") {
				pattern := strings.TrimPrefix(command, "? ")
				printGiveHelp(pl, pattern)
				return false
			}
			before := pl.Inventory.PeekItem(pl.HotbarSlot)
			pl.GivePlayer(command)
			after := pl.Inventory.PeekItem(pl.HotbarSlot)
			if before != after {
				sendEquipmentChangeForHotbarSlot(world, pl)
			}
		}

		if strings.HasPrefix(message, "/save") {
			var lines []string
			data := level.ToPlayerData(pl)
			if err := level.SavePlayerData(world.WorldDir, pl.Username, data); err != nil {
				lines = append(lines, fmt.Sprintf("Failed to save player data: %v", err))
			} else {
				lines = append(lines, "Player data saved successfully.")
			}
			if err := level.SaveMcRegion(world, world.WorldDir); err != nil {
				lines = append(lines, fmt.Sprintf("Failed to save mcr region: %v", err))
			} else {
				lines = append(lines, "MCR region saved successfully.")
			}
			sendDebugMessage(pl, lines...)
		}

		if strings.HasPrefix(message, "/destroy") {
			var x, y, z int32
			_, err := fmt.Sscanf(message, "/destroy %d %d %d", &x, &y, &z)
			if err != nil {
				sendUsage(pl, "/destroy")
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
				sendUsage(pl, "/gamemode")
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
			case "/kill", "":
				p.Message = "\u00A7c" + fmt.Sprintf("Server killed %s", pl.Username)
				world.BroadcastPacket(p.Serialize())
				sendSetHealth(pl.Connection, 0)
				pl.SetHP(0)
			default:
				sendUsage(pl, "/kill")
				return false
			}
		}

		if strings.HasPrefix(message, "/time") {
			var subcommand string
			var value int64
			_, err := fmt.Sscanf(message, "/time %s %d", &subcommand, &value)
			if err != nil {
				sendUsage(pl, "/time")
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
				sendUsage(pl, "/time")
			}
		}

		if strings.HasPrefix(message, "/debug") {
			command := strings.TrimPrefix(message, "/debug ")
			switch command {
			case "players":
				entities := world.SnapshotEntities()
				lines := []string{"Players:"}
				for _, e := range entities {
					if e.IsPlayer() {
						x, y, z := e.GetPosition()
						lines = append(lines, fmt.Sprintf("  [%d] HP=%d at x=%.2f, y=%.2f, z=%.2f", e.GetEntityId(), e.GetHP(), x, y, z))
					}
				}
				sendDebugMessage(pl, lines...)

			case "furnaces":
				for key, f := range world.Containers.Furnaces {
					sendDebugMessage(pl, fmt.Sprintf("Furnace x=%d, y=%d, z=%d:", key.X, key.Y, key.Z))
					for i, item := range f.Items {
						sendDebugMessage(pl, fmt.Sprintf("  slot %d: id=%d count=%d meta=%d", i, item.TypeId, item.Count, item.Metadata))
					}
				}
			case "water":
				chunks := world.LoadChunks()
				loadedSources := make(map[level.BlockKey]byte)
				loadedFlowing := make(map[level.BlockKey]byte)
				for _, chunk := range chunks {
					logic := chunk.Logic
					for key, height := range logic.WaterSources {
						loadedSources[key] = height
					}
					for key, height := range logic.FlowingWater {
						loadedFlowing[key] = height
					}
				}
				lines := []string{fmt.Sprintf("Water sources in world: %d", len(loadedSources))}
				for key := range loadedSources {
					lines = append(lines, fmt.Sprintf("  source at x=%d, y=%d, z=%d", key.X, key.Y, key.Z))
				}
				for key := range loadedFlowing {
					lines = append(lines, fmt.Sprintf("  flowing at x=%d, y=%d, z=%d", key.X, key.Y, key.Z))
				}
				sendDebugMessage(pl, lines...)
			case "fallables":
				loadedFallables := make(map[level.BlockKey]struct{})
				chunks := world.LoadChunks()
				for _, chunk := range chunks {
					logic := chunk.Logic
					for key, fallable := range logic.Fallables {
						loadedFallables[key] = fallable
					}
				}
				lines := []string{fmt.Sprintf("Falling blocks in world: %d", len(loadedFallables))}
				for key := range loadedFallables {
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
				loadedGrowables := make(map[level.BlockKey]level.Growable)
				chunks := world.LoadChunks()
				for _, chunk := range chunks {
					logic := chunk.Logic
					for key, growable := range logic.Growables {
						loadedGrowables[key] = growable
					}
				}
				lines := []string{fmt.Sprintf("Growables in world: %d", len(loadedGrowables))}
				for key, e := range loadedGrowables {
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
			case "block":
				pl.DebugBlock = !pl.DebugBlock
				lines := []string{fmt.Sprintf("Debug block mode: %t", pl.DebugBlock)}
				sendDebugMessage(pl, lines...)
			default:
				sendUsage(pl, "/debug")
			}
		}

		if strings.HasPrefix(message, "/help") {
			handleHelpCommand(pl, message)
		}

		return true
	}
	p.Message = "<" + pl.Username + "> " + message
	world.BroadcastPacket(p.Serialize())
	return false
}

func printGiveHelp(pl *player.Player, pattern string) {
	var re *regexp.Regexp
	if pattern != "" {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			sendDebugMessage(pl, fmt.Sprintf("Invalid pattern '%s', falling back to substring match: %v", pattern, err))
		} else {
			re = compiled
		}
	}

	matches := func(key string) bool {
		if pattern == "" {
			return true
		}
		if re != nil {
			return re.MatchString(key)
		}
		return strings.Contains(key, pattern)
	}

	blockKeys := make([]string, 0, len(constants.BlockCommandMap))
	for k := range constants.BlockCommandMap {
		if matches(k) {
			blockKeys = append(blockKeys, k)
		}
	}
	sort.Strings(blockKeys)

	itemKeys := make([]string, 0, len(constants.ItemMap))
	for k := range constants.ItemMap {
		if matches(k) {
			itemKeys = append(itemKeys, k)
		}
	}
	sort.Strings(itemKeys)

	var lines []string
	if pattern != "" {
		lines = append(lines, fmt.Sprintf("Matches for '%s':", pattern))
	}
	lines = append(lines, "Available blocks:")
	if len(blockKeys) == 0 {
		lines = append(lines, "  (none)")
	} else {
		lines = append(lines, blockKeys...)
	}
	lines = append(lines, "Available items:")
	if len(itemKeys) == 0 {
		lines = append(lines, "  (none)")
	} else {
		lines = append(lines, itemKeys...)
	}

	sendDebugMessage(pl, lines...)
}
