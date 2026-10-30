package packethandler

import (
	"math"
	"math/rand"
	"time"

	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
	"github.com/leNicDev/retromc/player"
)

const voidGlitchTime = 90 * time.Second

type voidGlitch struct {
	chunks player.ChunkSet
	target *[3]float64
}

var voidGlitches = map[*player.Player]*voidGlitch{}

func withheldChunk(pl *player.Player, x, z int32) bool {
	g := voidGlitches[pl]
	return g != nil && g.chunks.Has(x, z)
}

func hauntVoid(w *level.World, pl *player.Player) bool {
	if pl.Dimension != 0 || voidGlitches[pl] != nil {
		return false
	}
	cx, cz := level.WorldToChunkCoord(floorI(pl.X)), level.WorldToChunkCoord(floorI(pl.Z))
	yaw := float64(pl.Yaw) * math.Pi / 180
	var ahead player.ChunkCoord
	for _, dist := range []float64{12, 20, 28} {
		ahead = player.ChunkCoord{
			X: level.WorldToChunkCoord(floorI(pl.X - math.Sin(yaw)*dist)),
			Z: level.WorldToChunkCoord(floorI(pl.Z + math.Cos(yaw)*dist)),
		}
		if ahead.X != cx || ahead.Z != cz {
			break
		}
	}
	picks := []player.ChunkCoord{ahead}
	if rand.Intn(2) == 0 {
		side := [][2]int32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}[rand.Intn(4)]
		if c := (player.ChunkCoord{X: ahead.X + side[0], Z: ahead.Z + side[1]}); c.X != cx || c.Z != cz {
			picks = append(picks, c)
		}
	}
	g := &voidGlitch{chunks: player.ChunkSet{}}
	pl.SentChunksMu.Lock()
	for _, c := range picks {
		if !pl.SentChunks.Has(c.X, c.Z) {
			continue
		}
		unload := packets.SetChunkVisibilityPacket{X: c.X, Z: c.Z, Mode: false}
		pl.Connection.Write(unload.Serialize())
		delete(pl.SentChunks, c)
		delete(pl.PendingChunks, c)
		g.chunks.Set(c.X, c.Z)
	}
	pl.SentChunksMu.Unlock()
	if len(g.chunks) == 0 {
		return false
	}
	voidGlitches[pl] = g
	time.AfterFunc(voidGlitchTime, func() {
		w.Enqueue(func() {
			if voidGlitches[pl] != g {
				return
			}
			if !w.HasPlayer(pl) {
				delete(voidGlitches, pl)
			} else if g.target == nil {
				releaseVoid(w, pl, g)
			}
		})
	})
	return true
}

func releaseVoid(w *level.World, pl *player.Player, g *voidGlitch) {
	delete(voidGlitches, pl)
	pl.SentChunksMu.Lock()
	defer pl.SentChunksMu.Unlock()
	for c := range g.chunks {
		if pl.Dimension != 0 || !inViewDistance(c, pl.LastChunkX, pl.LastChunkZ) || pl.SentChunks.Has(c.X, c.Z) {
			continue
		}
		if chunk, ok := w.PeekChunk(c.X, c.Z, 0); ok {
			sendChunkToPlayer(w, pl, level.ChunkCoord{X: c.X, Z: c.Z}, 0, chunk)
		} else {
			pl.PendingChunks.Set(c.X, c.Z)
		}
	}
}

func columnTop(w *level.World, x, z, dim int32) int32 {
	y := int32(level.CHUNK_SIZE_Y - 1)
	for y > 0 && !entities.IsNormalCube(w.GetBlock(x, byte(y), z, dim)) {
		y--
	}
	return y + 1
}

func catchVoidFall(w *level.World, pl *player.Player) bool {
	g := voidGlitches[pl]
	if g == nil || pl.Dimension != 0 || pl.Transferring {
		return false
	}
	bx, by, bz := floorI(pl.X), floorI(pl.Y), floorI(pl.Z)
	if g.target == nil {
		if !g.chunks.Has(level.WorldToChunkCoord(bx), level.WorldToChunkCoord(bz)) || by >= columnTop(w, bx, bz, 0) {
			return false
		}
		cave := landing{minY: netherEntryY + 2, maxY: by - 2, padY: by - 2, covered: true}
		x, y, z, ok := findLanding(w, 0, bx, bz, cave)
		if !ok {
			x, y, z, ok = findLanding(w, 0, bx, bz, overworldLanding)
		}
		if !ok {
			x, y, z = bx, overworldLanding.padY, bz
			w.BuildLandingPad(x, y, z, 0, overworldLanding.floor)
		}
		g.target = &[3]float64{float64(x) + 0.5, float64(y) + 0.01, float64(z) + 0.5}
	}
	t := *g.target
	if pl.Y > t[1]+0.5 && pl.Y > netherEntryY+2 {
		return false
	}
	releaseVoid(w, pl, g)
	relocatePlayer(w, pl, t[0], t[1], t[2])
	return true
}

func fellThroughWorld(w *level.World, pl *player.Player) bool {
	bx, by, bz := floorI(pl.X), floorI(pl.Y), floorI(pl.Z)
	if !pl.SentChunks.Has(level.WorldToChunkCoord(bx), level.WorldToChunkCoord(bz)) {
		return true
	}
	if by < 0 || by+1 >= level.CHUNK_SIZE_Y {
		return false
	}
	return entities.IsNormalCube(w.GetBlock(bx, byte(by), bz, 0)) || entities.IsNormalCube(w.GetBlock(bx, byte(by+1), bz, 0))
}

func rescueToSurface(w *level.World, pl *player.Player) {
	bx, bz := floorI(pl.X), floorI(pl.Z)
	x, y, z, ok := findLanding(w, 0, bx, bz, overworldLanding)
	if !ok {
		x, y, z = bx, overworldLanding.padY, bz
		w.BuildLandingPad(x, y, z, 0, overworldLanding.floor)
	}
	relocatePlayer(w, pl, float64(x)+0.5, float64(y)+0.01, float64(z)+0.5)
}

func relocatePlayer(w *level.World, pl *player.Player, x, y, z float64) {
	cx, cz := level.WorldToChunkCoord(floorI(x)), level.WorldToChunkCoord(floorI(z))
	pl.SentChunksMu.Lock()
	for dx := int32(-1); dx <= 1; dx++ {
		for dz := int32(-1); dz <= 1; dz++ {
			c := level.ChunkCoord{X: cx + dx, Z: cz + dz}
			if pl.SentChunks.Has(c.X, c.Z) || withheldChunk(pl, c.X, c.Z) {
				continue
			}
			if chunk, ok := w.PeekChunk(c.X, c.Z, pl.Dimension); ok {
				sendChunkToPlayer(w, pl, c, pl.Dimension, chunk)
			}
		}
	}
	pl.SentChunksMu.Unlock()
	pl.SetPosition(x, y, z)
	pl.Stance = y + playerEyeHeight
	pl.FallDistance = 0
	pl.Transferring = true
	pl.AwaitingTeleportAck = true
	BroadcastTeleportPlayer(w, pl, x, y, z, byte(int32(math.Floor(float64(pl.Yaw)*256/360))))
	updateChunks(w, x, z, pl)
}
