package level

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"slices"
)

type ClaimBox struct {
	Min [3]int32 `json:"min"`
	Max [3]int32 `json:"max"`
	Dim int32    `json:"dim,omitempty"`
}

func (c *ClaimBox) UnmarshalJSON(raw []byte) error {
	var v struct {
		Min, Max     *[3]int32
		X, Y, Z, Dim int32
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	if v.Min == nil || v.Max == nil {
		v.Min, v.Max = &[3]int32{v.X, v.Y, v.Z}, &[3]int32{v.X, v.Y, v.Z}
	}
	*c = ClaimBox{Min: *v.Min, Max: *v.Max, Dim: v.Dim}
	return nil
}

func (c ClaimBox) contains(x, y, z, dim int32) bool {
	return c.Dim == dim && x >= c.Min[0] && x <= c.Max[0] && y >= c.Min[1] && y <= c.Max[1] && z >= c.Min[2] && z <= c.Max[2]
}

func (c ClaimBox) intersects(o ClaimBox) bool {
	return c.Dim == o.Dim && c.Min[0] <= o.Max[0] && o.Min[0] <= c.Max[0] && c.Min[1] <= o.Max[1] && o.Min[1] <= c.Max[1] && c.Min[2] <= o.Max[2] && o.Min[2] <= c.Max[2]
}

type Permissions struct {
	CanClaim     bool `json:"canClaim"`
	CanTime      bool `json:"canTime"`
	CanSave      bool `json:"canSave"`
	CanKill      bool `json:"canKill"`
	CanTpOthers  bool `json:"canTpOthers"`
	CanSummon    bool `json:"canSummon"`
	CanBan       bool `json:"canBan"`
	CanTp        bool `json:"canTp"`
	CanGive      bool `json:"canGive"`
	CanDestroy   bool `json:"canDestroy"`
	CanFill      bool `json:"canFill"`
	CanGamemode  bool `json:"canGamemode"`
	CanDim       bool `json:"canDim"`
	CanSize      bool `json:"canSize"`
	CanDebug     bool `json:"canDebug"`
	CanSkywalk   bool `json:"canSkywalk"`
	CanChainmine bool `json:"canChainmine"`
	CanHaunt     bool `json:"canHaunt"`
}

func (w *World) PublicCommands() []string {
	cfg := struct {
		Public []string `json:"public"`
	}{Public: []string{"stats"}}
	raw, err := os.ReadFile(filepath.Join(w.WorldDir, "commands.json"))
	if err != nil {
		return cfg.Public
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		log.Println("Failed to read commands.json, using defaults:", err)
		return []string{"stats"}
	}
	return cfg.Public
}

type Account struct {
	Permissions
	Claims []ClaimBox `json:"claims"`
}

func (w *World) accountsPath() string {
	return filepath.Join(w.WorldDir, "players.json")
}

func (w *World) readAccounts() (map[string]*Account, error) {
	accounts := make(map[string]*Account)
	raw, err := os.ReadFile(w.accountsPath())
	if os.IsNotExist(err) {
		return accounts, nil
	}
	if err != nil {
		return nil, err
	}
	return accounts, json.Unmarshal(raw, &accounts)
}

func (w *World) LoadAccounts() error {
	accounts, err := w.readAccounts()
	if err != nil {
		return err
	}
	w.accounts = accounts
	return nil
}

func (w *World) refreshPermissions() bool {
	disk, err := w.readAccounts()
	if err != nil {
		log.Println("Failed to read players.json, keeping it untouched:", err)
		return false
	}
	for name, acc := range w.accounts {
		acc.Permissions = Permissions{}
		if d, ok := disk[name]; ok {
			acc.Permissions = d.Permissions
		}
	}
	for name, d := range disk {
		if _, ok := w.accounts[name]; !ok {
			w.accounts[name] = &Account{Permissions: d.Permissions, Claims: []ClaimBox{}}
		}
	}
	return true
}

func (w *World) saveAccounts() {
	if !w.refreshPermissions() {
		return
	}
	data, err := json.MarshalIndent(w.accounts, "", "  ")
	if err != nil {
		log.Println("Failed to encode players.json:", err)
		return
	}
	tmp := w.accountsPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		log.Println("Failed to write players.json:", err)
		return
	}
	if err := os.Rename(tmp, w.accountsPath()); err != nil {
		log.Println("Failed to write players.json:", err)
	}
}

func (w *World) RegisterAccount(name string) {
	if _, ok := w.accounts[name]; ok {
		return
	}
	w.accounts[name] = &Account{Claims: []ClaimBox{}}
	w.saveAccounts()
}

func (w *World) PermissionsOf(name string) Permissions {
	w.refreshPermissions()
	if acc, ok := w.accounts[name]; ok {
		return acc.Permissions
	}
	return Permissions{}
}

func (w *World) ClaimOwner(x, y, z, dim int32) (string, bool) {
	for name, acc := range w.accounts {
		for _, c := range acc.Claims {
			if c.contains(x, y, z, dim) {
				return name, true
			}
		}
	}
	return "", false
}

func (w *World) IsClaimed(x, y, z, dim int32) bool {
	_, ok := w.ClaimOwner(x, y, z, dim)
	return ok
}

func (w *World) TouchesClaim(x, y, z, dim int32) bool {
	area := ClaimBox{Min: [3]int32{x - 1, y - 1, z - 1}, Max: [3]int32{x + 1, y + 1, z + 1}, Dim: dim}
	for _, acc := range w.accounts {
		for _, c := range acc.Claims {
			if c.intersects(area) {
				return true
			}
		}
	}
	return false
}

func (w *World) Claim(name string, box ClaimBox) {
	acc := w.accounts[name]
	acc.Claims = append(acc.Claims, box)
	w.saveAccounts()
}

func (w *World) Unclaim(area ClaimBox) int {
	removed := 0
	for _, acc := range w.accounts {
		n := len(acc.Claims)
		acc.Claims = slices.DeleteFunc(acc.Claims, area.intersects)
		removed += n - len(acc.Claims)
	}
	if removed > 0 {
		w.saveAccounts()
	}
	return removed
}
