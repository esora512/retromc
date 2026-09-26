package level

import (
	"math"

	"github.com/leNicDev/retromc/constants"
	"github.com/leNicDev/retromc/entities"
	"github.com/leNicDev/retromc/mcregion"
)

func (w *World) CaptureEntities(chunks map[ChunkCoord]*Chunk, dim int32, onRemove func(id int32)) map[ChunkCoord][]*mcregion.Compound {
	out := make(map[ChunkCoord][]*mcregion.Compound)
	for coord, ch := range chunks {
		if ch != nil && ch.HadEntities {
			out[coord] = nil
		}
	}

	for id, e := range w.Entities {
		if e.GetDim() != dim || e.GetEntityType() == constants.Player {
			continue
		}
		x, _, z := e.GetPosition()
		coord := ChunkCoord{WorldToChunkCoord(int32(math.Floor(x))), WorldToChunkCoord(int32(math.Floor(z)))}
		if ch, ok := chunks[coord]; !ok || ch == nil {
			continue
		}
		if nbt := entities.EntityToNBT(e); nbt != nil {
			out[coord] = append(out[coord], nbt)
		}
		if onRemove != nil {
			delete(w.Entities, id)
			w.BroadcastPacket(w.DespawnEntity(id))
			onRemove(id)
		}
	}

	for coord, list := range out {
		chunks[coord].HadEntities = len(list) > 0
	}
	return out
}

func (w *World) spawnPendingEntities(c *Chunk, dim int32) {
	if len(c.PendingEntities) == 0 {
		return
	}
	for _, t := range c.PendingEntities {
		if e := entities.EntityFromNBT(t, w.NextEntityId(), dim); e != nil {
			w.Entities[e.GetEntityId()] = e
		}
	}
	c.PendingEntities = nil
	c.HadEntities = true
}
