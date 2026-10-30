package packethandler

import (
	"fmt"
	"math"
	"math/rand"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/inventory"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

var HauntingEnabled = true

const (
	day        = 24000
	hauntDelay = 3 * day
)

// a player's haunting stage is reached after this much play time; stage 0 is left alone
var stageStart = [...]int64{0, hauntDelay, hauntDelay + 2*day, hauntDelay + 5*day}

// average ticks between two random haunts of the same player, per stage
var hauntEvery = [...]int{0, 20 * 60 * 25, 20 * 60 * 15, 20 * 60 * 10}

const (
	effectDoor   = 1003
	effectRecord = 1005
	effectSmoke  = 2000
	effectBreak  = 2001
	record13     = 2256
)

type hauntEvent struct {
	name   string
	stage  int
	weight int
	run    func(w *level.World, pl *player.Player) bool
}

var hauntEvents = []hauntEvent{
	{"appear", 1, 8, hauntAppear},
	{"cave", 1, 4, hauntCave},
	{"door", 1, 3, hauntDoor},
	{"steps", 1, 3, hauntSteps},
	{"torches", 1, 2, hauntTorches},
	{"sign", 1, 2, hauntSign},
	{"whisper", 1, 1, hauntWhisper},
	{"disc", 1, 1, hauntDisc},
	{"rain", 1, 1, hauntRain},
	{"dusk", 1, 1, hauntDusk},
	{"behind", 2, 4, hauntBehind},
	{"bed", 2, 3, hauntBed},
	{"possess", 2, 3, hauntPossess},
	{"void", 2, 2, hauntVoid},
	{"gift", 2, 2, hauntGift},
	{"lightning", 2, 2, hauntLightning},
	{"join", 2, 2, hauntJoin},
	{"pyramid", 2, 1, hauntPyramid},
	{"stalk", 3, 3, hauntStalk},
	{"crash", 3, 1, hauntCrash},
}

var (
	hauntedRain = map[int32]bool{}
	hauntedDusk = map[int32]int64{}
	lastCrash   = map[string]time.Time{}
	ghosts      = map[int32][]*ghostSet{}
)

func hauntStage(pl *player.Player) int {
	for s := len(stageStart) - 1; s > 0; s-- {
		if pl.HauntTicks >= stageStart[s] {
			return s
		}
	}
	return 0
}

func TickHaunting(w *level.World, tracker *entities.EntityTracker) {
	for _, e := range w.Entities {
		m, ok := e.(*entities.Mob)
		if !ok {
			continue
		}
		if m.IsHerobrine() && m.Vanished {
			vanishHerobrine(w, tracker, m)
		}
		if m.DiedPossessed {
			m.DiedPossessed = false
			bolt := packets.ThunderboltPacket{EntityId: w.NextEntityId(), X: floorI(m.X), Y: floorI(m.Y), Z: floorI(m.Z)}
			w.BroadcastPacketInDim(m.Dimension, bolt.Serialize())
		}
	}
	for _, pl := range w.Players {
		if !pl.LoggedIn || pl.HP <= 0 {
			continue
		}
		pl.HauntTicks++
		if st := hauntStage(pl); HauntingEnabled && st > 0 && rand.Intn(hauntEvery[st]) == 0 {
			randomHaunt(w, pl, st)
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

func randomHaunt(w *level.World, pl *player.Player, stage int) {
	var events []hauntEvent
	total := 0
	for _, e := range hauntEvents {
		if e.stage <= stage {
			events = append(events, e)
			total += e.weight
		}
	}
	for range 3 {
		r := rand.Intn(total)
		for _, e := range events {
			if r -= e.weight; r < 0 {
				if e.run(w, pl) {
					return
				}
				break
			}
		}
	}
}

func hauntEventList() []string {
	lines := []string{"Usage: /haunt <player> <event>"}
	for st := 1; st < len(stageStart); st++ {
		var names []string
		for _, e := range hauntEvents {
			if e.stage == st {
				names = append(names, e.name)
			}
		}
		lines = append(lines, fmt.Sprintf("Stage %d: %s", st, strings.Join(names, ", ")))
	}
	return lines
}

func handleHauntCommand(pl *player.Player, world *level.World, args []string) {
	sendDebugMessage(pl, HauntCommand(world, args)...)
}

// HauntCommand runs /haunt with the given arguments and returns the reply lines
func HauntCommand(world *level.World, args []string) []string {
	if len(args) == 0 {
		return []string{"Usage: " + commandUsage("/haunt")}
	}
	switch args[0] {
	case "on", "off":
		HauntingEnabled = args[0] == "on"
		return []string{fmt.Sprintf("Random haunting %s", map[bool]string{true: "enabled", false: "disabled"}[HauntingEnabled])}
	case "list", "events":
		return hauntEventList()
	}

	target := world.GetFirstPlayerByName(args[0])
	if target == nil || !target.LoggedIn {
		return []string{fmt.Sprintf("Player not found: %s", args[0])}
	}
	if len(args) == 1 {
		randomHaunt(world, target, max(hauntStage(target), 1))
		return []string{fmt.Sprintf("Haunting %s", target.Username)}
	}
	if args[1] == "stage" {
		if len(args) > 2 {
			st, err := strconv.Atoi(args[2])
			if err != nil || st < 0 || st >= len(stageStart) {
				return []string{fmt.Sprintf("Stage must be 0-%d", len(stageStart)-1)}
			}
			target.HauntTicks = stageStart[st]
		}
		return []string{fmt.Sprintf("%s is at haunting stage %d", target.Username, hauntStage(target))}
	}
	for _, e := range hauntEvents {
		if e.name == args[1] {
			if e.run(world, target) {
				return []string{fmt.Sprintf("Haunting %s: %s", target.Username, e.name)}
			}
			return []string{fmt.Sprintf("Could not run %s for %s right now", e.name, target.Username)}
		}
	}
	return append([]string{"Unknown event: " + args[1]}, hauntEventList()...)
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
	_, ok := w.SpawnHerobrine(pl, 26, entities.HerobrineWatch)
	return ok
}

func hauntCave(w *level.World, pl *player.Player) bool {
	if !underground(w, pl) {
		return false
	}
	_, ok := w.SpawnHerobrine(pl, 14, entities.HerobrineWatch)
	return ok
}

func hauntBehind(w *level.World, pl *player.Player) bool {
	x, z := behind(pl, 3+rand.Float64()*2)
	y, ok := w.GroundY(x, z, floorI(pl.Y), pl.Dimension)
	if !ok {
		return false
	}
	m, ok := w.SpawnHerobrineAt(pl, x, y, z, entities.HerobrineLurk)
	if ok && !m.InSightOf(w, pl, -1) {
		w.RemoveEntity(m.EntityId)
		return false
	}
	return ok
}

func hauntBed(w *level.World, pl *player.Player) bool {
	return bedSighting(w, pl, true)
}

// bedSighting puts him at the foot of the player's bed
func bedSighting(w *level.World, pl *player.Player, unseen bool) bool {
	if !pl.HasBedSpawn || pl.Dimension != 0 {
		return false
	}
	hx, hy, hz := pl.BedSpawnX, int32(pl.BedSpawnY), pl.BedSpawnZ
	if dx, dz := float64(hx)-pl.X, float64(hz)-pl.Z; dx*dx+dz*dz > 16*16 {
		return false
	}
	head := w.GetBlock(hx, byte(hy), hz, 0)
	if !head.IsBedHead() {
		return false
	}
	d := [4][2]int32{{0, 1}, {-1, 0}, {0, -1}, {1, 0}}[head.Metadata&3]
	fx, fz := hx-d[0], hz-d[1]
	spots := [][2]int32{{fx - d[0], fz - d[1]}, {fx - d[1], fz + d[0]}, {fx + d[1], fz - d[0]}}
	for _, s := range spots {
		m, ok := w.SpawnHerobrineAt(pl, s[0], hy, s[1], entities.HerobrineLurk)
		if !ok {
			continue
		}
		if unseen && m.InSightOf(w, pl, 0.5) {
			w.RemoveEntity(m.EntityId)
			return false
		}
		return true
	}
	return false
}

// OnWake sometimes has him waiting at the foot of the bed in the morning
func OnWake(w *level.World, pl *player.Player) {
	if !HauntingEnabled || hauntStage(pl) < 2 || rand.Intn(3) != 0 {
		return
	}
	later(w, 1500*time.Millisecond, pl, func() {
		bedSighting(w, pl, false)
	})
}

// hauntDeniesSleep keeps the player awake while he is around
func hauntDeniesSleep(w *level.World, pl *player.Player) bool {
	if !HauntingEnabled || hauntStage(pl) < 2 {
		return false
	}
	if _, near := w.Herobrine(pl.GetEntityId()); !near {
		if rand.Intn(4) != 0 {
			return false
		}
		hauntBehind(w, pl)
	}
	sendChat(pl, "You may not rest now, there are monsters nearby")
	return true
}

func hauntStalk(w *level.World, pl *player.Player) bool {
	if pl.IsCreative {
		return false
	}
	_, ok := w.SpawnHerobrine(pl, 20, entities.HerobrineStalk)
	return ok
}

// hauntCrash stares the player in the face, then drops the connection so the client shows a genuine "Connection Lost"
func hauntCrash(w *level.World, pl *player.Player) bool {
	if time.Since(lastCrash[pl.Username]) < 30*time.Minute {
		return false
	}
	lastCrash[pl.Username] = time.Now()
	yaw := float64(pl.Yaw) * math.Pi / 180
	x, z := floorI(pl.X-math.Sin(yaw)*1.8), floorI(pl.Z+math.Cos(yaw)*1.8)
	w.SpawnHerobrineAt(pl, x, floorI(pl.Y), z, entities.HerobrineScare)
	later(w, 800*time.Millisecond, pl, func() {
		pl.Connection.Close()
	})
	return true
}

var gifts = []struct {
	id    int16
	count byte
}{
	{constants.RedstoneTorchOn.Value, 1},
	{constants.Rose.Value, 1},
	{constants.MossyCobblestone.Value, 1},
	{constants.Netherrack.Value, 1},
	{constants.Sign.Value, 1},
	{constants.Bone.Value, 3},
	{constants.Porkchop.Value, 1},
	{constants.Redstone.Value, 4},
	{constants.Feather.Value, 2},
	{constants.Cookie.Value, 1},
}

// hauntGift leaves something in one of the chests near the player, never taking anything
func hauntGift(w *level.World, pl *player.Player) bool {
	const r = 24
	var chests []*inventory.Chest
	for k, ch := range w.Containers.Chests {
		if k.Dim != pl.Dimension || slices.Contains(chests, ch) || chestViewed(w, ch) || !slices.ContainsFunc(ch.Items, emptySlot) {
			continue
		}
		if dx, dy, dz := float64(k.X)-pl.X, float64(k.Y)-pl.Y, float64(k.Z)-pl.Z; dx*dx+dy*dy+dz*dz > r*r {
			continue
		}
		if owner, claimed := w.ClaimOwner(k.X, int32(k.Y), k.Z, k.Dim); claimed && owner != pl.Username {
			continue
		}
		chests = append(chests, ch)
	}
	if len(chests) == 0 {
		return false
	}
	ch := chests[rand.Intn(len(chests))]
	g := gifts[rand.Intn(len(gifts))]
	ch.Items[slices.IndexFunc(ch.Items, emptySlot)] = inventory.NewItem(g.id, g.count, 0)
	return true
}

func emptySlot(it inventory.Item) bool { return it.TypeId == -1 }

func chestViewed(w *level.World, ch *inventory.Chest) bool {
	for _, p := range w.Players {
		if p.LoggedIn && p.InventoryType == player.ChestInventory && w.GetChest(p.Chest.X, p.Chest.Y, p.Chest.Z, p.Chest.Dim) == ch {
			return true
		}
	}
	return false
}

func hauntPossess(w *level.World, pl *player.Player) bool {
	const r = 24
	var animals []*entities.Mob
	for _, e := range w.Entities {
		m, ok := e.(*entities.Mob)
		if !ok || !m.IsPassive() || m.IsPossessed() || m.HP <= 0 || m.Dimension != pl.Dimension {
			continue
		}
		if dx, dy, dz := m.X-pl.X, m.Y-pl.Y, m.Z-pl.Z; dx*dx+dy*dy+dz*dz <= r*r {
			animals = append(animals, m)
		}
	}
	if len(animals) == 0 {
		return false
	}
	animals[rand.Intn(len(animals))].Possess(pl)
	return true
}

func isShrine(w *level.World, x, y, z, dim int32) bool {
	is := func(dx, dy, dz int32, types ...int16) bool {
		return slices.Contains(types, int16(w.GetBlock(x+dx, byte(y+dy), z+dz, dim).TypeId))
	}
	if !is(0, 0, 0, constants.Netherrack.Value) || !is(0, -1, 0, constants.MossyCobblestone.Value, constants.GoldBlock.Value) {
		return false
	}
	for dx := int32(-1); dx <= 1; dx++ {
		for dz := int32(-1); dz <= 1; dz++ {
			if (dx != 0 || dz != 0) && !is(dx, -1, dz, constants.GoldBlock.Value) {
				return false
			}
		}
	}
	for _, n := range [][2]int32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if !is(n[0], 0, n[1], constants.RedstoneTorchOn.Value, constants.RedstoneTorchOff.Value) {
			return false
		}
	}
	return true
}

// lightShrine answers a lit shrine with lightning and skips the player's grace period
func lightShrine(w *level.World, pl *player.Player, x, y, z int32) {
	if !HauntingEnabled || !isShrine(w, x, y, z, pl.Dimension) {
		return
	}
	bolt := packets.ThunderboltPacket{EntityId: w.NextEntityId(), X: x, Y: y + 1, Z: z}
	w.BroadcastPacketInDim(pl.Dimension, bolt.Serialize())
	pl.HauntTicks = max(pl.HauntTicks, hauntDelay)
	later(w, 3*time.Second, pl, func() {
		hauntJoin(w, pl)
	})
}

func underground(w *level.World, pl *player.Player) bool {
	if pl.Y > 60 {
		return false
	}
	x, z := floorI(pl.X), floorI(pl.Z)
	for y := floorI(pl.Y) + 2; y < level.WorldMaxY; y++ {
		if b := w.GetBlock(x, byte(y), z, pl.Dimension); b.IsSolid() {
			return true
		}
	}
	return false
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

type ghostSet struct {
	dim    int32
	blocks map[[3]int32]constants.WBlock
}

// ghostBlocks shows blocks to a single player only and puts the real ones back after a while
func ghostBlocks(w *level.World, pl *player.Player, blocks map[[3]int32]constants.WBlock, restoreAfter time.Duration) {
	for pos, b := range blocks {
		p := packets.SetBlockPacket{X: pos[0], Y: byte(pos[1]), Z: pos[2], BlockType: b.TypeId, BlockMeta: b.Metadata}
		pl.Connection.Write(p.Serialize())
	}
	g := &ghostSet{dim: pl.Dimension, blocks: blocks}
	id := pl.GetEntityId()
	ghosts[id] = append(ghosts[id], g)
	time.AfterFunc(restoreAfter, func() {
		w.Enqueue(func() { dissolveGhost(w, pl, g) })
	})
}

func dissolveGhost(w *level.World, pl *player.Player, g *ghostSet) {
	id := pl.GetEntityId()
	i := slices.Index(ghosts[id], g)
	if i < 0 {
		return
	}
	if ghosts[id] = slices.Delete(ghosts[id], i, i+1); len(ghosts[id]) == 0 {
		delete(ghosts, id)
	}
	if !online(w, pl) || pl.Dimension != g.dim {
		return
	}
	for pos := range g.blocks {
		sendBlockToPlayer(w, pl, pos[0], byte(pos[1]), pos[2], w.GetBlock(pos[0], byte(pos[1]), pos[2], g.dim))
	}
}

// touchGhost makes a ghost structure vanish as soon as the player digs or clicks any part of it
func touchGhost(w *level.World, pl *player.Player, x, y, z int32) bool {
	for _, g := range ghosts[pl.GetEntityId()] {
		if _, ok := g.blocks[[3]int32{x, y, z}]; ok && g.dim == pl.Dimension {
			dissolveGhost(w, pl, g)
			sendWorldEvent(pl, effectSmoke, x, y, z, 4)
			return true
		}
	}
	return false
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
