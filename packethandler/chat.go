package packethandler

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"fmt"

	"runtime"
	"slices"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	entPack "github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/inventory"
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
	{"/give", "/give <item> [amount]"},
	{"/save", "/save"},
	{"/destroy", "/destroy <x> <y> <z>"},
	{"/fill", "/fill | /fill <block> [<x1> <y1> <z1> <x2> <y2> <z2>]"},
	{"/gamemode", "/gamemode <0|1>"},
	{"/kill", "/kill [entities]"},
	{"/time", "/time <set | speed> <value> | /time freeze"},
	{"/debug", "/debug <water | entities | time | block | furnaces>"},
	{"/help", "/help [command]"},
	{"/tp", "/tp <x> <y> <z> | tp <p1> <p2>"},
	{"/size", "/size"},
	{"/version", "/version"},
	{"/stats", "/stats"},
	{"/summon", "/summon [zombie|skeleton|spider|creeper|pig|sheep] [x y z]"},
	{"/chainmine", "/chainmine"},
	{"/skywalk", "/skywalk"},
	{"/hollow", "/hollow | /hollow <block> [<x1> <y1> <z1> <x2> <y2> <z2>]"},
	{"/claim", "/claim | /claim <add | remove> [<x1> <y1> <z1> <x2> <y2> <z2>]"},
	{"/ban", "/ban <player>"},
	{"/clear", "/clear"},
	{"/haunt", "/haunt <on | off | list> | /haunt <player> [event | stage [0-3]]"},
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

func permitted(world *level.World, pl *player.Player, message string) bool {
	if cmd := strings.TrimPrefix(strings.Fields(message)[0], "/"); slices.Contains(world.PublicCommands(), cmd) {
		return true
	}
	perms := world.PermissionsOf(pl.Username)
	switch {
	case strings.HasPrefix(message, "/claim"):
		return perms.CanClaim
	case strings.HasPrefix(message, "/time"):
		return perms.CanTime
	case strings.HasPrefix(message, "/save"):
		return perms.CanSave
	case strings.HasPrefix(message, "/summon"):
		return perms.CanSummon
	case strings.HasPrefix(message, "/kill"):
		return perms.CanKill || strings.TrimPrefix(message, "/kill ") != "entities"
	case strings.HasPrefix(message, "/ban"):
		return perms.CanBan
	case strings.HasPrefix(message, "/tp"):
		args := strings.Fields(strings.TrimPrefix(message, "/tp"))
		if len(args) == 2 && args[0] != pl.Username {
			return perms.CanTpOthers
		}
		return perms.CanTp
	case strings.HasPrefix(message, "/give"):
		return perms.CanGive
	case strings.HasPrefix(message, "/destroy"):
		return perms.CanDestroy
	case strings.HasPrefix(message, "/fill"), strings.HasPrefix(message, "/hollow"):
		return perms.CanFill
	case strings.HasPrefix(message, "/gamemode"):
		return perms.CanGamemode
	case strings.HasPrefix(message, "/dim"):
		return perms.CanDim
	case strings.HasPrefix(message, "/size"):
		return perms.CanSize
	case strings.HasPrefix(message, "/debug"):
		return perms.CanDebug
	case strings.HasPrefix(message, "/skywalk"):
		return perms.CanSkywalk
	case strings.HasPrefix(message, "/chainmine"):
		return perms.CanChainmine
	case strings.HasPrefix(message, "/haunt"):
		return perms.CanHaunt
	}
	return true
}

func handleChatMessageInPacket(p packets.ChatMessagePacket, pl *player.Player, world *level.World, tracker *entities.EntityTracker) bool {
	if !pl.LoggedIn {
		return false
	}
	message := p.Message
	if pl.AuthPending {
		handleAuthCommand(pl, world, message)
		return false
	}
	if strings.HasPrefix(message, "/") {
		cmdName := strings.Fields(message)[0]
		if !permitted(world, pl, message) {
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
				pl.Immune = 200
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
				if pl1.Dimension != pl2.Dimension {
					transferPlayer(world, pl1, pl2.Dimension, x, y, z)
					return false
				}
				pl1.SetPosition(x, y, z)
				pl1.Immune = 200
				BroadcastTeleportPlayer(world, pl1, x, y, z, byte(pl1.Yaw))
				return false

			default:
				sendUsage(pl, "/tp")
				return false
			}
		}

		if strings.HasPrefix(message, "/dim") {
			if pl.Dimension == 0 {
				transferToNether(world, pl)
			} else {
				transferToOverworld(world, pl)
			}
		}

		if cmdName == "/fill" || cmdName == "/hollow" {
			handleFillCommand(pl, world, cmdName, strings.Fields(message)[1:])
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

		if strings.HasPrefix(message, "/stats") {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			mb := func(b uint64) float64 { return float64(b) / 1024 / 1024 }
			mobs := 0
			for _, e := range world.SnapshotEntities() {
				if e.GetEntityType() != constants.Player {
					mobs++
				}
			}
			sendDebugMessage(pl,
				fmt.Sprintf("Memory used: %.1f MB (heap %.1f MB)", mb(m.Sys-m.HeapReleased), mb(m.HeapAlloc)),
				fmt.Sprintf("Reserved from OS: %.1f MB", mb(m.Sys)),
				fmt.Sprintf("Players: %d", world.OnlineCount()),
				fmt.Sprintf("Entities: %d", mobs),
			)
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
				lines = append(lines, "MCR region save queued.")
			}
			if world.HasManualBackup() {
				world.TriggerManualBackup()
				lines = append(lines, "B2 backup upload queued.")
			} else {
				lines = append(lines, "B2 backup not configured.")
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
			world.BreakContainer(x, y, z, world.GetBlock(x, byte(y), z, pl.Dimension).TypeId, pl.Dimension)
			world.SetBlockInQueue(x, y, z, air, pl.Dimension)
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
				pl.Skywalk = true 
				p.Message = fmt.Sprintf("%s switched to Creative mode", pl.Username)
				world.BroadcastPacket(p.Serialize())
				sendDebugMessage(pl, "Skywalk enabled")
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
				world.RelayEvent(pl.Username, "Server killed "+pl.Username)
				SendSetHealth(pl.Connection, 0)
				pl.SetHP(0)
				killPlayer(world, pl)
			default:
				sendUsage(pl, "/kill")
				return false
			}
		}

		if strings.HasPrefix(message, "/summon") {
			parts := strings.Fields(message)
			x, y, z := int32(pl.X), int32(pl.Y), int32(pl.Z)

			mobType := "spider"
			coordIdx := 1
			if len(parts) >= 2 {
				if _, err := strconv.Atoi(parts[1]); err != nil {
					mobType = strings.ToLower(parts[1])
					coordIdx = 2
				}
			}

			if len(parts) >= coordIdx+3 {
				px, errX := strconv.Atoi(parts[coordIdx])
				py, errY := strconv.Atoi(parts[coordIdx+1])
				pz, errZ := strconv.Atoi(parts[coordIdx+2])

				if errX == nil && errY == nil && errZ == nil {
					x, y, z = int32(px), int32(py), int32(pz)
				}
			}

			mobTypes := map[string]byte{
				"zombie": constants.Zombie, "skeleton": constants.Skeleton, "spider": constants.Spider,
				"creeper": constants.Creeper, "pig": constants.Pig, "sheep": constants.Sheep,
			}
			t, ok := mobTypes[mobType]
			if !ok {
				sendUsage(pl, "/summon")
				return false
			}
			world.SpawnMobType(t, x, y, z, pl.Dimension, -1)
			sendDebugMessage(pl, fmt.Sprintf("Spawned %s at x=%d, y=%d, z=%d", mobType, x, y, z))
		}

		if cmdName == "/claim" {
			handleClaimCommand(pl, world, strings.Fields(message)[1:])
			return false
		}

		if cmdName == "/clear" {
			pl.Inventory = inventory.NewInventory(player.PLAYER_INVENTORY_SIZE)
			sendInventory(pl.Connection, pl, world)
			sendEquipmentChangeForHotbarSlot(world, pl)
			sendDebugMessage(pl, "Inventory cleared")
			return false
		}

		if cmdName == "/ban" {
			handleBanCommand(pl, world, strings.Fields(message)[1:])
			return false
		}

		if cmdName == "/haunt" {
			handleHauntCommand(pl, world, strings.Fields(message)[1:])
			return false
		}

		if strings.HasPrefix(message, "/skywalk") {
			pl.Skywalk = !pl.Skywalk
			pl.SkywalkFalling = !pl.Skywalk
			pl.SkywalkY, pl.SkywalkDim = int32(math.Floor(pl.Y))-1, pl.Dimension
			updateSkywalk(world, pl)
			state := "disabled"
			if pl.Skywalk {
				state = "enabled"
			}
			sendDebugMessage(pl, fmt.Sprintf("Skywalk %s", state))
		}

		if strings.HasPrefix(message, "/chainmine") {
			pl.ChainMiningEnabled = !pl.ChainMiningEnabled
			state := "disabled"
			if pl.ChainMiningEnabled {
				state = "enabled"
			}
			sendDebugMessage(pl, fmt.Sprintf("Chain mining %s", state))
		}

		if message == "/time freeze" {
			world.Frozen = !world.Frozen
			if world.Frozen {
				BroadcastWorldMsg(world, pl.Username+" froze time")
			} else {
				BroadcastWorldMsg(world, pl.Username+" unfroze time")
			}
			return false
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
			case "speed":
				if value < 1 || value > 20 {
					sendDebugMessage(pl, "Speed must be between 1 (normal) and 20")
					return false
				}
				world.Speed = value
				sendDebugMessage(pl, fmt.Sprintf("Speed set to %d", value))
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
					if e.GetEntityType() == constants.Player {
						x, y, z := e.GetPosition()
						lines = append(lines, fmt.Sprintf("  [%d] HP=%d at x=%.2f, y=%.2f, z=%.2f, isRiding=%d", e.GetEntityId(), e.GetHP(), x, y, z, pl.IsRiding))
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
	// clients drop the connection on chat lines over 119 chars
	if r := []rune(message); len(r) > 100 {
		message = string(r[:100])
	}
	p.Message = "<" + pl.Username + "> " + message
	world.BroadcastPacket(p.Serialize())
	world.RelayChat(pl.Username, message)
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
