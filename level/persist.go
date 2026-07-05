package level

import (
	"encoding/gob"
	"log"
	"os"
)

type BlockRecord struct {
	X        int32
	Y        byte
	Z        int32
	Type     byte
	Meta     byte
	Light    byte
	SkyLight byte
}

func (w *World) SaveChanges(path string) error {
	w.Mu.RLock()
	records := make([]BlockRecord, 0, len(w.changes))
	for k, b := range w.changes {
		records = append(records, BlockRecord{
			X:        k.X,
			Y:        k.Y,
			Z:        k.Z,
			Type:     b.TypeId,
			Meta:     b.Metadata,
			Light:    b.Light,
			SkyLight: b.SkyLight,
		})
	}
	w.Mu.RUnlock()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	log.Println("Save completed")
	return gob.NewEncoder(f).Encode(records)
}

func (w *World) LoadChanges(path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		log.Println("No world save found, starting fresh.")
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	var records []BlockRecord
	if err := gob.NewDecoder(f).Decode(&records); err != nil {
		return err
	}

	w.Mu.Lock()
	for _, r := range records {
		w.changes[BlockKey{r.X, r.Y, r.Z}] = Block{
			TypeId:   r.Type,
			Metadata: r.Meta,
			Light:    r.Light,
			SkyLight: r.SkyLight,
		}
	}
	w.Mu.Unlock()

	log.Printf("Loaded %d block changes from %s", len(records), path)
	chunksToLoad := make(map[ChunkCoord]struct{})
	for k := range w.changes {
		cx := WorldToChunkCoord(k.X)
		cz := WorldToChunkCoord(k.Z)
		chunksToLoad[ChunkCoord{cx, cz}] = struct{}{}
	}
	for coord := range chunksToLoad {
		w.GetOrCreateChunk(coord.X, coord.Z, w.WorldType)
	}
	return nil
}
