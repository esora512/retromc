package packethandler

import (
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/player"
)

const (
	dreamStride     = 10.0
	dreamShiftAhead = 20.0
	dreamCooldown   = 12 * time.Second
	dreamWakeY      = -8.0
)

type dreamer struct {
	ax, az, hx, hz float64
	heading        bool
	nextShift      time.Time
}

var dreamers = map[*player.Player]*dreamer{}

var shiftWhispers = []string{
	"The tube forgets.",
	"Something moved behind you.",
	"It was different a moment ago.",
	"The dream shifts.",
	"You hear wool rustling.",
}

func otherDreamers(w *level.World, pl *player.Player) []*player.Player {
	var out []*player.Player
	w.ForEachPlayer(func(p *player.Player) {
		if p != pl && p.Dimension == level.DreamDim {
			out = append(out, p)
		}
	})
	return out
}

func dreamEntry(w *level.World, pl *player.Player) (float64, float64, float64) {
	others := otherDreamers(w, pl)
	if len(others) == 0 {
		w.ResetDream()
		return w.DreamSpawn()
	}
	host := others[rand.Intn(len(others))]
	return host.X, host.Y, host.Z
}

func placeInDream(w *level.World, pl *player.Player) {
	pl.Dimension = level.DreamDim
	pl.X, pl.Y, pl.Z = dreamEntry(w, pl)
	pl.Stance = pl.Y + playerEyeHeight
	pl.FallDistance = 0
	delete(dreamers, pl)
}

func enterDream(w *level.World, pl *player.Player) {
	x, y, z := dreamEntry(w, pl)
	sendChat(pl, "§7§oYou drift off...")
	transferPlayer(w, pl, level.DreamDim, x, y, z)
}

func wakeFromDream(w *level.World, pl *player.Player) {
	delete(dreamers, pl)
	sendChat(pl, "§7§oYou wake up.")
	transferToOverworld(w, pl)
}

func dreamStep(w *level.World, pl *player.Player) bool {
	if pl.Y < dreamWakeY {
		wakeFromDream(w, pl)
		return true
	}
	d := dreamers[pl]
	if d == nil {
		dreamers[pl] = &dreamer{ax: pl.X, az: pl.Z}
		return false
	}
	dx, dz := pl.X-d.ax, pl.Z-d.az
	dist := math.Hypot(dx, dz)
	if dist < dreamStride {
		return false
	}
	hx, hz := dx/dist, dz/dist
	if d.heading && hx*d.hx+hz*d.hz < -0.5 && time.Now().After(d.nextShift) && rand.Intn(5) != 0 {
		shiftDream(w, pl, hx, hz)
		d.nextShift = time.Now().Add(dreamCooldown)
	}
	d.ax, d.az, d.hx, d.hz, d.heading = pl.X, pl.Z, hx, hz, true
	return false
}

func shiftDream(w *level.World, pl *player.Player, hx, hz float64) {
	px, pz := floorI(pl.X), floorI(pl.Z)
	ocx := level.WorldToChunkCoord(floorI(pl.X + hx*96))
	ocz := level.WorldToChunkCoord(floorI(pl.Z + hz*96))
	scene := w.DreamShiftScene(int(px), int(pz), math.Abs(hz) > math.Abs(hx), ocx, ocz)

	watchers := append(otherDreamers(w, pl), pl)
	var coords []level.ChunkCoord
	for _, c := range w.DreamChunkCoords() {
		ox, oz := float64(c.X*16+8)-pl.X, float64(c.Z*16+8)-pl.Z
		if ox*hx+oz*hz < dreamShiftAhead || math.Hypot(ox, oz) > level.VIEW_DISTANCE*16 {
			continue
		}
		occupied := false
		for _, p := range watchers {
			pcx, pcz := level.WorldToChunkCoord(floorI(p.X)), level.WorldToChunkCoord(floorI(p.Z))
			if abs32(c.X-pcx) <= 1 && abs32(c.Z-pcz) <= 1 {
				occupied = true
				break
			}
		}
		if !occupied {
			coords = append(coords, c)
		}
	}
	if len(coords) == 0 {
		return
	}
	sort.Slice(coords, func(i, j int) bool {
		return chunkDistSq(coords[i], pl) < chunkDistSq(coords[j], pl)
	})
	w.AssignDreamScene(coords, scene)
	for _, c := range coords {
		w.RequestChunkAsync(c.X, c.Z, level.DreamDim, func(ch *level.Chunk) {
			w.Enqueue(func() {
				if w.ReplaceDreamChunk(c, ch) {
					resendDreamChunk(w, c)
				}
			})
		})
	}
	if rand.Intn(3) == 0 {
		sendChat(pl, "§8§o"+shiftWhispers[rand.Intn(len(shiftWhispers))])
	}
}

func chunkDistSq(c level.ChunkCoord, pl *player.Player) float64 {
	dx, dz := float64(c.X*16+8)-pl.X, float64(c.Z*16+8)-pl.Z
	return dx*dx + dz*dz
}

func resendDreamChunk(w *level.World, c level.ChunkCoord) {
	w.ForEachPlayer(func(p *player.Player) {
		if p.Dimension != level.DreamDim {
			return
		}
		p.SentChunksMu.Lock()
		defer p.SentChunksMu.Unlock()
		if !p.SentChunks.Has(c.X, c.Z) {
			return
		}
		if p.PendingChunks == nil {
			p.PendingChunks = make(player.ChunkSet)
		}
		delete(p.SentChunks, player.ChunkCoord{X: c.X, Z: c.Z})
		p.PendingChunks.Set(c.X, c.Z)
	})
}

func DreamCommand(w *level.World, args []string) []string {
	if len(args) != 1 {
		return []string{"Usage: dream <player>"}
	}
	var pl *player.Player
	w.ForEachPlayer(func(p *player.Player) {
		if strings.EqualFold(p.Username, args[0]) {
			pl = p
		}
	})
	switch {
	case pl == nil:
		return []string{args[0] + " is not online"}
	case pl.Dimension == level.DreamDim:
		return []string{pl.Username + " is already dreaming"}
	case pl.AuthPending || pl.Transferring || pl.HP <= 0:
		return []string{pl.Username + " can't be moved right now, try again"}
	}
	enterDream(w, pl)
	return []string{"Sent " + pl.Username + " into the dream"}
}

func TickDream(w *level.World, tracker *entities.EntityTracker) {
	w.TrimDream(tracker.ResetEntity)
	for pl := range dreamers {
		if !w.HasPlayer(pl) || pl.Dimension != level.DreamDim {
			delete(dreamers, pl)
		}
	}
}
