package packethandler

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"fmt"

	"runtime"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
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

var commandHelp = []struct {
	Name  string
	Usage string
}{
	{"/give", "/give <item>"},
	{"/save", "/save"},
	{"/destroy", "/destroy <x> <y> <z>"},
	{"/place", "/place <block> <x> <y> <z> | /place fill <block> <x1> <z1> <x2> <z2> <y>"},
	{"/gamemode", "/gamemode <0|1>"},
	{"/kill", "/kill [entities]"},
	{"/time", "/time <set | tickspeed> <value>"},
	{"/debug", "/debug <water | fallables | entities | growables | time | block | furnaces>"},
	{"/help", "/help [command]"},
	{"/tp", "/tp <x> <y> <z> | tp <p1> <p2>"},
	{"/size", "/size"},
	{"/version", "/version"},
	{"/summon", "/summon [x y z]"},
}

var opOnlyCommands = map[string]bool{
	"/give":     true,
	"/save":     true,
	"/destroy":  true,
	"/place":    true,
	"/gamemode": true,
	"/kill":     true,
	"/time":     true,
	"/summon":   true,
	"/tp":       true,
	"/dim":      true,
}

var debugSubcommandRequiresOp = map[string]bool{
	"entities": true,
}

func sendDebugMessage(pl *player.Player, lines ...string) {
	for _, line := range lines {
		p := packets.ChatMessagePacket{Message: "\u00A7e" + line}
		pl.Connection.Write(p.Serialize())
	}
}

func sendNoPermission(pl *player.Player, name string) {
	sendDebugMessage(pl, fmt.Sprintf("\u00A7cYou do not have permission to use %s.", name))
}

func BroadcastWorldMsg(w *level.World, msg string) {
	p := packets.ChatMessagePacket{Message: "\u00A7e" + msg}
	w.BroadcastPacket(p.Serialize())
}

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

func handleChatMessageInPacket(p packets.ChatMessagePacket, pl *player.Player, world *level.World, tracker *entities.EntityTracker) bool {
	message := p.Message
	if strings.HasPrefix(message, "/") {
		// Determine the base command token (e.g. "/tp" from "/tp 1 2 3")
		// and reject it up-front if it's op-only and the player isn't an op.
		cmdName := strings.Fields(message)[0]
		if opOnlyCommands[cmdName] && !pl.IsOp {
			sendNoPermission(pl, cmdName)
			return false
		}

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
				pl.Immune = 0
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
				pl.Immune = 0
				BroadcastTeleportPlayer(world, pl1, x, y, z, byte(pl1.Yaw))
				return false

			default:
				sendUsage(pl, "/tp")
				return false
			}
		}

		if strings.HasPrefix(message, "/dim") {
			pl.X = player.SpawnX
			pl.Y = player.SpawnY
			pl.Z = player.SpawnZ
			pl.Stance = player.SpawnStance
			pl.Yaw = 0
			pl.Pitch = 0
			pl.OnGround = true
			loc := int32(0)
			if pl.Dimension == loc {
				loc = -1
				pl.Dimension = -1
			} else {
				pl.Dimension = loc
			}

			pl.SentChunks = make(player.ChunkSet)
			pl.HasInitializedChunks = false

			pl.Immune = 0

			sendRespawn(pl.Connection, byte(loc))

			pl.SetHP(20)
			SendSetHealth(pl.Connection, 20.0)
			sendPlayerPositionAndLook(pl.Connection, 0, 0, 80)
			world.MulticastPacket(packets.NewAddPassengerPacket(pl.GetEntityId(), -1), pl)
			world.MulticastPacket(packets.NewTeleportPlayerPacket(pl, pl.X, pl.Y, pl.Z, float64(pl.Yaw), float64(pl.Pitch), world), pl)
		}

		if strings.HasPrefix(message, "/place") {
			args := strings.Fields(strings.TrimPrefix(message, "/place"))
			if len(args) == 0 {
				sendUsage(pl, "/place")
				return false
			}

			if args[0] == "fill" {
				handlePlaceFillCommand(pl, world, args[1:])
				return false
			}
			// Expected format: /place <block> <x> <y> <z>
			if len(args) != 4 {
				sendUsage(pl, "/place")
				return false
			}

			blockName := args[0]
			var x, y, z int32
			_, err := fmt.Sscanf(strings.Join(args[1:], " "), "%d %d %d", &x, &y, &z)
			if err != nil {
				sendUsage(pl, "/place")
				return false
			}

			b := constants.GetBlockByName(blockName)
			if b.Value == -1 {
				sendDebugMessage(pl, fmt.Sprintf("Unknown block: %s", blockName))
				return false
			}
			block := constants.NewBlockById(b.Value, byte(b.Meta))

			world.SetBlockInQueue(x, y, z, block, pl.Dimension)
			sendDebugMessage(pl, fmt.Sprintf("Placed %s at x=%d, y=%d, z=%d", blockName, x, y, z))
			return false
		}

		if strings.HasPrefix(message, "/size") {
			sendDebugMessage(pl, fmt.Sprintf("World size = %s", world.SizeString()))
			sendDebugMessage(pl, fmt.Sprintf("OChunks = %d", len(world.LoadChunks(0))))
			sendDebugMessage(pl, fmt.Sprintf("NChunks = %d", len(world.LoadChunks(-1))))

			alloc, sys, totalAlloc, numGC := LogMemStats()
			sendDebugMessage(pl, alloc)
			sendDebugMessage(pl, sys)
			sendDebugMessage(pl, totalAlloc)
			sendDebugMessage(pl, numGC)
		}

		if strings.HasPrefix(message, "/version") {
			sendDebugMessage(pl, fmt.Sprintf("dev/%s", world.CommitHash))
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
			air := constants.NewAirBlock()
			world.SetBlock(x, byte(y), z, air, pl.Dimension)
			blockChange := packets.SetBlockPacket{
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
					if e.GetEntityType() != constants.Player {
						e.SetHP(0)
					}
				}
			case "/kill", "":
				p.Message = "\u00A7c" + fmt.Sprintf("Server killed %s", pl.Username)
				world.BroadcastPacket(p.Serialize())
				SendSetHealth(pl.Connection, 0)
				pl.SetHP(0)
				pl.DespawnIn = 21
				DropInventory(world, &pl.Inventory, pl.X, pl.Y, pl.Z, pl.GetDim(), tracker)
			default:
				sendUsage(pl, "/kill")
				return false
			}
		}

		if strings.HasPrefix(message, "/summon") {
			parts := strings.Fields(message)
			x, y, z := int32(pl.X), int32(pl.Y), int32(pl.Z)

			if len(parts) >= 4 {
				px, errX := strconv.Atoi(parts[1])
				py, errY := strconv.Atoi(parts[2])
				pz, errZ := strconv.Atoi(parts[3])

				if errX == nil && errY == nil && errZ == nil {
					x, y, z = int32(px), int32(py), int32(pz)
				}
			}
			sendDebugMessage(pl, fmt.Sprintf("Spawned Spider at x=%d, y=%d, z=%d", x, y, z))
			world.SpawnSpider(x, y, z, pl.Dimension, -1)
		}

		if strings.HasPrefix(message, "/time") {
			var subcommand string
			var value int64
			_, err := fmt.Sscanf(message, "/time %s %d", &subcommand, &value)
			if err != nil {
				// TODO: Change this to be more elegant...
				sendDebugMessage(pl, fmt.Sprintf("Current Time is %d", world.TimeTick))
				return false
			}
			switch subcommand {
			case "set":
				world.TimeTick = value
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
			// Extract the subcommand token (ignore any trailing args) so we
			// can check permissions even if a subcommand ever takes args.
			subcommand := strings.Fields(command)
			sub := ""
			if len(subcommand) > 0 {
				sub = subcommand[0]
			}
			if debugSubcommandRequiresOp[sub] && !pl.IsOp {
				sendNoPermission(pl, "/debug "+sub)
				return false
			}

			switch command {
			case "players":
				entities := world.SnapshotEntities()
				lines := []string{"Players:"}
				for _, e := range entities {
					if e.GetEntityType() == constants.Player {
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
				chunks := world.LoadChunks(0)
				chunks = append(chunks, world.LoadChunks(-1)...)
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

type chunkChanges struct {
	coords []uint16
	types  []byte
	meta   []byte
}

func formatMultiBlock(localX, y, localZ int32) uint16 {
	return uint16((localX&0xF)<<12 |
		(localZ&0xF)<<8 |
		(y & 0xFF))
}

func handlePlaceFillCommand(pl *player.Player, world *level.World, args []string) {
	if len(args) != 6 {
		sendUsage(pl, "/place")
		return
	}

	blockName := args[0]
	b := constants.GetBlockByName(blockName)
	if b.Value == -1 {
		sendDebugMessage(pl, fmt.Sprintf("Unknown block: %s", blockName))
		return
	}

	var x1, z1, x2, z2, y int32
	_, err := fmt.Sscanf(
		strings.Join(args[1:], " "),
		"%d %d %d %d %d",
		&x1, &z1, &x2, &z2, &y,
	)
	if err != nil {
		sendUsage(pl, "/place")
		return
	}

	minX, maxX := x1, x2
	if minX > maxX {
		minX, maxX = maxX, minX
	}

	minZ, maxZ := z1, z2
	if minZ > maxZ {
		minZ, maxZ = maxZ, minZ
	}

	block := constants.NewBlockById(b.Value, byte(b.Meta))

	changes := make(map[[2]int32]*chunkChanges)

	placed := 0

	for x := minX; x <= maxX; x++ {
		for z := minZ; z <= maxZ; z++ {

			world.SetBlock(x, byte(y), z, block, pl.Dimension)

			chunkX := level.WorldToChunkCoord(x)
			chunkZ := level.WorldToChunkCoord(z)
			key := [2]int32{chunkX, chunkZ}

			change, ok := changes[key]
			if !ok {
				change = &chunkChanges{}
				changes[key] = change
			}

			change.coords = append(change.coords,
				formatMultiBlock(x&0xF, y, z&0xF))

			change.types = append(change.types, byte(block.TypeId))
			change.meta = append(change.meta, block.Metadata)

			placed++
		}
	}

	for key, change := range changes {
		p := packets.SetMultipleBlocksPacket{
			ChunkX:      key[0],
			ChunkZ:      key[1],
			NumOfBlocks: uint16(len(change.coords)),
			BlockCoords: change.coords,
			BlockTypes:  change.types,
			Metadata:    change.meta,
		}
		world.BroadcastPacket(p.Serialize())
	}
	sendDebugMessage(
		pl,
		fmt.Sprintf(
			"Placed %d %s block(s) in filled area at y=%d",
			placed,
			blockName,
			y,
		),
	)
}
