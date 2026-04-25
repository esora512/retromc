package player

import (
	"encoding/gob"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/leNicDev/retromc/inventory"
)

const playerSaveDir = "saves"

// sanitizeUsername strips path separators and other characters that should
// never appear in a filename. Minecraft usernames are alphanumeric plus a
// few symbols, so this is a defensive guard rather than a normaliser.
func sanitizeUsername(username string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		"..", "_",
		":", "_",
	)
	return replacer.Replace(username)
}

func playerSavePath(username string) string {
	return filepath.Join(playerSaveDir, sanitizeUsername(username)+".dat")
}

// SaveInventory writes the player's inventory to disk under players/<username>.dat.
func SaveInventory(username string, inv inventory.Inventory) error {
	if username == "" {
		return errors.New("cannot save inventory: empty username")
	}
	if err := os.MkdirAll(playerSaveDir, 0o755); err != nil {
		return err
	}
	f, err := os.Create(playerSavePath(username))
	if err != nil {
		return err
	}
	defer f.Close()
	if err := gob.NewEncoder(f).Encode(inv.Items); err != nil {
		return err
	}
	log.Printf("Saved inventory for %s", username)
	return nil
}

// LoadInventory loads the player's inventory from disk into inv.
// Returns (true, nil) on success, (false, nil) when no save exists.
func LoadInventory(username string, inv *inventory.Inventory) (bool, error) {
	if username == "" {
		return false, errors.New("cannot load inventory: empty username")
	}
	f, err := os.Open(playerSavePath(username))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()

	var items []inventory.Item
	if err := gob.NewDecoder(f).Decode(&items); err != nil {
		return false, err
	}
	if len(items) != len(inv.Items) {
		log.Printf("Inventory size mismatch for %s (saved %d, expected %d); ignoring save",
			username, len(items), len(inv.Items))
		return false, nil
	}
	copy(inv.Items, items)
	log.Printf("Loaded inventory for %s", username)
	return true, nil
}
