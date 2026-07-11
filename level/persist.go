package level

import (
	"encoding/gob"
	"log"
	"os"

	"github.com/leNicDev/retromc/inventory"
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

type containerKind byte

const (
	kindChest containerKind = iota
	kindFurnace
	kindDispenser
)

type containerRecord struct {
	Kind                      containerKind
	X, Y, Z                   int32
	Size                      uint16
	HasSecond                 bool
	SecondX, SecondY, SecondZ int32
	Items                     []inventory.Item
}

// SaveContainers writes all chest, furnace and dispenser contents to disk.
func (w *World) SaveContainers(path string) error {
	var records []containerRecord

	// Chests: dedupe double chests (both positions point at the same struct).
	seen := make(map[*inventory.Chest]bool)
	for _, c := range w.GetAllChests() {
		if seen[c] {
			continue
		}
		seen[c] = true
		rec := containerRecord{
			Kind:  kindChest,
			X:     c.Position.X,
			Y:     c.Position.Y,
			Z:     c.Position.Z,
			Size:  c.Size,
			Items: c.Items,
		}
		if c.Size == DOUBLE_CHEST_SIZE {
			rec.HasSecond = true
			rec.SecondX = c.SecondPosition.X
			rec.SecondY = c.SecondPosition.Y
			rec.SecondZ = c.SecondPosition.Z
		}
		records = append(records, rec)
	}

	for _, f := range w.GetAllFurnaces() {
		records = append(records, containerRecord{
			Kind:  kindFurnace,
			X:     f.Position.X,
			Y:     f.Position.Y,
			Z:     f.Position.Z,
			Size:  f.Size,
			Items: f.Items,
		})
	}

	for _, d := range w.GetAllDispensers() {
		records = append(records, containerRecord{
			Kind:  kindDispenser,
			X:     d.Position.X,
			Y:     d.Position.Y,
			Z:     d.Position.Z,
			Size:  d.Size,
			Items: d.Items,
		})
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := gob.NewEncoder(f).Encode(records); err != nil {
		return err
	}
	log.Printf("Saved %d containers to %s", len(records), path)
	return nil
}

func (w *World) LoadContainers(path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		log.Println("No container save found, starting fresh.")
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	var records []containerRecord
	if err := gob.NewDecoder(f).Decode(&records); err != nil {
		return err
	}

	for _, rec := range records {
		switch rec.Kind {
		case kindChest:
			w.PlaceChest(rec.X, rec.Y, rec.Z)
			if rec.HasSecond {
				w.PlaceChest(rec.SecondX, rec.SecondY, rec.SecondZ)
			}
			chest := w.GetChest(rec.X, rec.Y, rec.Z)
			if chest != nil && len(rec.Items) == len(chest.Items) {
				copy(chest.Items, rec.Items)
			}
		case kindFurnace:
			w.PlaceFurnace(rec.X, rec.Y, rec.Z)
			furnace := w.GetFurnace(rec.X, rec.Y, rec.Z)
			if furnace != nil && len(rec.Items) == len(furnace.Items) {
				copy(furnace.Items, rec.Items)
			}
		case kindDispenser:
			w.PlaceDispenser(rec.X, rec.Y, rec.Z)
			dispenser := w.GetDispenser(rec.X, rec.Y, rec.Z)
			if dispenser != nil && len(rec.Items) == len(dispenser.Items) {
				copy(dispenser.Items, rec.Items)
			}
		}
	}

	log.Printf("Loaded %d containers from %s", len(records), path)
	return nil
}
