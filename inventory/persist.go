package inventory

import (
	"encoding/gob"
	"log"
	"os"
)

type containerKind byte

const (
	kindChest containerKind = iota
	kindFurnace
	kindDispenser
)

// containerRecord is the on-disk representation of a single container.
// Double chests are stored as a single record with HasSecond set.
type containerRecord struct {
	Kind                      containerKind
	X, Y, Z                   int32
	Size                      uint16
	HasSecond                 bool
	SecondX, SecondY, SecondZ int32
	Items                     []Item
}

// SaveContainers writes all chest, furnace and dispenser contents to disk.
func SaveContainers(path string) error {
	var records []containerRecord

	// Chests: dedupe double chests (both positions point at the same struct).
	seen := make(map[*Chest]bool)
	for _, c := range chestRegistry {
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

	for _, f := range FurnaceRegistry {
		records = append(records, containerRecord{
			Kind:  kindFurnace,
			X:     f.Position.X,
			Y:     f.Position.Y,
			Z:     f.Position.Z,
			Size:  f.Size,
			Items: f.Items,
		})
	}

	for _, d := range dispenserRegistry {
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

// LoadContainers restores chest, furnace and dispenser contents from disk.
// Returns nil (without error) when the file does not exist.
func LoadContainers(path string) error {
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
			PlaceChest(rec.X, rec.Y, rec.Z)
			if rec.HasSecond {
				PlaceChest(rec.SecondX, rec.SecondY, rec.SecondZ)
			}
			chest := GetChest(rec.X, rec.Y, rec.Z)
			if chest != nil && len(rec.Items) == len(chest.Items) {
				copy(chest.Items, rec.Items)
			}
		case kindFurnace:
			PlaceFurnace(rec.X, rec.Y, rec.Z)
			furnace := GetFurnace(rec.X, rec.Y, rec.Z)
			if furnace != nil && len(rec.Items) == len(furnace.Items) {
				copy(furnace.Items, rec.Items)
			}
		case kindDispenser:
			PlaceDispenser(rec.X, rec.Y, rec.Z)
			dispenser := GetDispenser(rec.X, rec.Y, rec.Z)
			if dispenser != nil && len(rec.Items) == len(dispenser.Items) {
				copy(dispenser.Items, rec.Items)
			}
		}
	}

	log.Printf("Loaded %d containers from %s", len(records), path)
	return nil
}
