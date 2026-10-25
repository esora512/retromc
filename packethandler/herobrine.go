package packethandler

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

var HauntingEnabled = true

// average ticks between two random haunts of the same player (~25 min)
const hauntEvery = 20 * 60 * 25

const (
	effectDoor   = 1003
	effectRecord = 1005
	effectSmoke  = 2000
	effectBreak  = 2001
	record13     = 2256
)

type hauntEvent struct {
	name   string
	weight int
	run    func(w *level.World, pl *player.Player) bool
}

var hauntEvents = []hauntEvent{
	{"appear", 8, hauntAppear},
	{"door", 3, hauntDoor},
	{"steps", 3, hauntSteps},
	{"torches", 2, hauntTorches},
	{"lightning", 2, hauntLightning},
	{"sign", 2, hauntSign},
	{"join", 2, hauntJoin},
	{"whisper", 1, hauntWhisper},
	{"disc", 1, hauntDisc},
	{"rain", 1, hauntRain},
	{"dusk", 1, hauntDusk},
	{"pyramid", 1, hauntPyramid},
}

var (
	hauntedRain = map[int32]bool{}
	hauntedDusk = map[int32]int64{}
)

func TickHaunting(w *level.World, tracker *entities.EntityTracker) {
	for _, e := range w.Entities {
		if m, ok := e.(*entities.Mob); ok && m.IsHerobrine() && m.Vanished {
			vanishHerobrine(w, tracker, m)
		}
	}
	if !HauntingEnabled {
		return
	}
	for _, pl := range w.Players {
		if pl.LoggedIn && pl.HP > 0 && rand.Intn(hauntEvery) == 0 {
			randomHaunt(w, pl)
		}
	}
}

func vanishHerobrine(w *level.World, tracker *entities.EntityTracker, m *entities.Mob) {
	if pl, ok := w.GetPlayer(m.TargetId); ok && pl.LoggedIn && pl.Dimension == m.Dimension {
		x, y, z := floorI(m.X), floorI(m.Y), floorI(m.Z)
		sendWorldEvent(pl, effectSmoke, x, y, z, 4)
		sendWorldEvent(pl, effectSmoke, x, y+1, z, 4)
	}
	tracker.SendToViewers(w, m.EntityId, w.DespawnEntity(m.EntityId))
	w.RemoveEntity(m.EntityId)
	tracker.ResetEntity(m.EntityId)
}

func randomHaunt(w *level.World, pl *player.Player) {
	total := 0
	for _, e := range hauntEvents {
		total += e.weight
	}
	for range 3 {
		r := rand.Intn(total)
		for _, e := range hauntEvents {
			if r -= e.weight; r < 0 {
				if e.run(w, pl) {
					return
				}
				break
			}
		}
	}
}

func hauntEventNames() string {
	names := make([]string, len(hauntEvents))
	for i, e := range hauntEvents {
		names[i] = e.name
	}
	return strings.Join(names, "|")
}

func handleHauntCommand(pl *player.Player, world *level.World, args []string) {
	if len(args) == 0 {
		sendUsage(pl, "/haunt")
		return
	}
	switch args[0] {
	case "on", "off":
		HauntingEnabled = args[0] == "on"
		sendDebugMessage(pl, fmt.Sprintf("Random haunting %s", map[bool]string{true: "enabled", false: "disabled"}[HauntingEnabled]))
		return
	case "events":
		sendDebugMessage(pl, "Events: "+hauntEventNames())
		return
	}

	target := world.GetFirstPlayerByName(args[0])
	if target == nil || !target.LoggedIn {
		sendDebugMessage(pl, fmt.Sprintf("Player not found: %s", args[0]))
		return
	}
	if len(args) == 1 {
		randomHaunt(world, target)
		sendDebugMessage(pl, fmt.Sprintf("Haunting %s", target.Username))
		return
	}
	for _, e := range hauntEvents {
		if e.name == args[1] {
			if e.run(world, target) {
				sendDebugMessage(pl, fmt.Sprintf("Haunting %s: %s", target.Username, e.name))
			} else {
				sendDebugMessage(pl, fmt.Sprintf("Could not run %s for %s right now", e.name, target.Username))
			}
			return
		}
	}
	sendDebugMessage(pl, "Events: "+hauntEventNames())
}

// hauntedTime is the time of day a haunted player's client is shown instead of the real one
func hauntedTime(w *level.World, pl *player.Player) (int64, bool) {
	until, ok := hauntedDusk[pl.GetEntityId()]
	if !ok {
		return 0, false
	}
	if w.Tick > until {
		delete(hauntedDusk, pl.GetEntityId())
		return 0, false
	}
	return w.TimeTick - w.TimeTick%24000 + 18000, true
}

func hauntAppear(w *level.World, pl *player.Player) bool {
	_, ok := w.SpawnHerobrine(pl)
	return ok
}

func hauntDoor(w *level.World, pl *player.Player) bool {
	x, z := behind(pl, 4)
	y := floorI(pl.Y) + 1
	sendWorldEvent(pl, effectDoor, x, y, z, 0)
	later(w, 1500*time.Millisecond, pl, func() {
		sendWorldEvent(pl, effectDoor, x, y, z, 0)
	})
	return true
}

func hauntSteps(w *level.World, pl *player.Player) bool {
	for i := range 5 {
		later(w, time.Duration(i)*450*time.Millisecond, pl, func() {
			x, z := behind(pl, 6-float64(i))
			y := floorI(pl.Y) - 1
			if b := w.GetBlock(x, byte(max(y, 0)), z, pl.Dimension); b.TypeId != 0 {
				sendWorldEvent(pl, effectBreak, x, y, z, int32(b.TypeId))
			}
		})
	}
	return true
}

func hauntTorches(w *level.World, pl *player.Player) bool {
	const r = 10
	px, py, pz := floorI(pl.X), floorI(pl.Y), floorI(pl.Z)
	var torches [][3]int32
	for x := px - r; x <= px+r; x++ {
		for y := max(py-r, 1); y <= min(py+r, level.WorldMaxY-1); y++ {
			for z := pz - r; z <= pz+r; z++ {
				if w.GetBlock(x, byte(y), z, pl.Dimension).TypeId == byte(constants.Torch.Value) {
					torches = append(torches, [3]int32{x, y, z})
				}
			}
		}
	}
	if len(torches) == 0 {
		return false
	}
	rand.Shuffle(len(torches), func(i, j int) { torches[i], torches[j] = torches[j], torches[i] })
	ghost := map[[3]int32]constants.WBlock{}
	for _, t := range torches[:min(len(torches), 8)] {
		ghost[t] = constants.NewAirBlock()
		sendWorldEvent(pl, effectSmoke, t[0], t[1], t[2], 4)
	}
	ghostBlocks(w, pl, ghost, 3*time.Minute)
	return true
}

func hauntLightning(w *level.World, pl *player.Player) bool {
	angle := rand.Float64() * 2 * math.Pi
	dist := 6 + rand.Float64()*6
	x, z := floorI(pl.X+math.Cos(angle)*dist), floorI(pl.Z+math.Sin(angle)*dist)
	y, ok := w.GroundY(x, z, floorI(pl.Y), pl.Dimension)
	if !ok {
		return false
	}
	p := packets.ThunderboltPacket{EntityId: w.NextEntityId(), X: x, Y: y, Z: z}
	pl.Connection.Write(p.Serialize())
	return true
}

var signTexts = [][4]string{
	{"", "I am", "watching", ""},
	{"", "Turn back", "", ""},
	{"", "Why do you", "stay here?", ""},
	{"", "", "Behind you", ""},
}

func hauntSign(w *level.World, pl *player.Player) bool {
	x, z := behind(pl, 3+rand.Float64()*3)
	y, ok := w.GroundY(x, z, floorI(pl.Y), pl.Dimension)
	if !ok || w.GetBlock(x, byte(y), z, pl.Dimension).TypeId != 0 {
		return false
	}
	lookYaw := math.Atan2(float64(z)+0.5-pl.Z, float64(x)+0.5-pl.X)*180/math.Pi - 90
	meta := byte(int32(math.Floor((lookYaw+180)*16/360+0.5)) & 15)
	ghostBlocks(w, pl, map[[3]int32]constants.WBlock{{x, y, z}: constants.NewBlockById(constants.SignGround.Value, meta)}, 2*time.Minute)
	lines := signTexts[rand.Intn(len(signTexts))]
	sign := packets.UpdateSignPacket{X: x, Y: int16(y), Z: z, Text1: lines[0], Text2: lines[1], Text3: lines[2], Text4: lines[3]}
	pl.Connection.Write(sign.Serialize())
	return true
}

func hauntJoin(w *level.World, pl *player.Player) bool {
	if _, real := w.GetPlayerByUsername(entities.HerobrineName); real {
		return false
	}
	sendChat(pl, "§e"+entities.HerobrineName+" joined the game")
	later(w, time.Duration(4+rand.Intn(6))*time.Second, pl, func() {
		pl.Connection.Write(NewLeftGameMsg(entities.HerobrineName))
	})
	return true
}

var whispers = []string{"I see you.", "Behind you.", "You are not alone.", "Leave.", "..."}

func hauntWhisper(w *level.World, pl *player.Player) bool {
	if _, real := w.GetPlayerByUsername(entities.HerobrineName); real {
		return false
	}
	sendChat(pl, "<"+entities.HerobrineName+"> "+whispers[rand.Intn(len(whispers))])
	return true
}

func hauntDisc(w *level.World, pl *player.Player) bool {
	x, z := behind(pl, 8)
	y := max(floorI(pl.Y)-6, 1)
	sendWorldEvent(pl, effectRecord, x, y, z, record13)
	later(w, time.Duration(25+rand.Intn(20))*time.Second, pl, func() {
		sendWorldEvent(pl, effectRecord, x, y, z, 0)
	})
	return true
}

func hauntRain(w *level.World, pl *player.Player) bool {
	id := pl.GetEntityId()
	if hauntedRain[id] {
		return false
	}
	hauntedRain[id] = true
	start := packets.NewStatePacket{Reason: packets.StateBeginRain}
	pl.Connection.Write(start.Serialize())
	time.AfterFunc(time.Duration(60+rand.Intn(60))*time.Second, func() {
		w.Enqueue(func() {
			delete(hauntedRain, id)
			if online(w, pl) {
				stop := packets.NewStatePacket{Reason: packets.StateEndRain}
				pl.Connection.Write(stop.Serialize())
			}
		})
	})
	return true
}

func hauntDusk(w *level.World, pl *player.Player) bool {
	if w.IsNight() {
		return false
	}
	hauntedDusk[pl.GetEntityId()] = w.Tick + 20*40
	return true
}

func hauntPyramid(w *level.World, pl *player.Player) bool {
	angle := rand.Float64() * 2 * math.Pi
	dist := 18 + rand.Float64()*10
	cx, cz := floorI(pl.X+math.Cos(angle)*dist), floorI(pl.Z+math.Sin(angle)*dist)
	cy, ok := w.GroundY(cx, cz, floorI(pl.Y), pl.Dimension)
	if !ok || cy+3 >= level.WorldMaxY {
		return false
	}
	sand := constants.NewBlockById(constants.Sand.Value, 0)
	ghost := map[[3]int32]constants.WBlock{}
	for layer := int32(0); layer < 3; layer++ {
		r := 2 - layer
		for dx := -r; dx <= r; dx++ {
			for dz := -r; dz <= r; dz++ {
				pos := [3]int32{cx + dx, cy + layer, cz + dz}
				if w.GetBlock(pos[0], byte(pos[1]), pos[2], pl.Dimension).TypeId == 0 {
					ghost[pos] = sand
				}
			}
		}
	}
	ghost[[3]int32{cx, cy + 3, cz}] = constants.NewBlockById(constants.RedstoneTorchOn.Value, 5)
	ghostBlocks(w, pl, ghost, 5*time.Minute)
	return true
}

// ghostBlocks shows blocks to a single player only and puts the real ones back after a while
func ghostBlocks(w *level.World, pl *player.Player, blocks map[[3]int32]constants.WBlock, restoreAfter time.Duration) {
	for pos, b := range blocks {
		p := packets.SetBlockPacket{X: pos[0], Y: byte(pos[1]), Z: pos[2], BlockType: b.TypeId, BlockMeta: b.Metadata}
		pl.Connection.Write(p.Serialize())
	}
	dim := pl.Dimension
	later(w, restoreAfter, pl, func() {
		if pl.Dimension != dim {
			return
		}
		for pos := range blocks {
			sendBlockToPlayer(w, pl, pos[0], byte(pos[1]), pos[2], w.GetBlock(pos[0], byte(pos[1]), pos[2], dim))
		}
	})
}

func later(w *level.World, d time.Duration, pl *player.Player, fn func()) {
	time.AfterFunc(d, func() {
		w.Enqueue(func() {
			if online(w, pl) {
				fn()
			}
		})
	})
}

func online(w *level.World, pl *player.Player) bool {
	cur, ok := w.GetPlayer(pl.GetEntityId())
	return ok && cur == pl && pl.LoggedIn
}

func behind(pl *player.Player, dist float64) (int32, int32) {
	yaw := float64(pl.Yaw) * math.Pi / 180
	return floorI(pl.X + math.Sin(yaw)*dist), floorI(pl.Z - math.Cos(yaw)*dist)
}

func sendWorldEvent(pl *player.Player, effect, x, y, z, data int32) {
	p := packets.WorldEventPacket{EffectId: effect, X: x, Y: byte(min(max(y, 0), 127)), Z: z, Data: data}
	pl.Connection.Write(p.Serialize())
}

func sendChat(pl *player.Player, msg string) {
	p := packets.ChatMessagePacket{Message: msg}
	pl.Connection.Write(p.Serialize())
}

func floorI(v float64) int32 {
	return int32(math.Floor(v))
}
