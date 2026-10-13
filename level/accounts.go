package level

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type ClaimPos struct {
	X   int32 `json:"x"`
	Y   int32 `json:"y"`
	Z   int32 `json:"z"`
	Dim int32 `json:"dim,omitempty"`
}

type Permissions struct {
	CanClaim    bool `json:"canClaim"`
	CanTime     bool `json:"canTime"`
	CanSave     bool `json:"canSave"`
	CanKill     bool `json:"canKill"`
	CanTpOthers bool `json:"canTpOthers"`
	CanSummon   bool `json:"canSummon"`
}

type Account struct {
	Permissions
	Claims []ClaimPos `json:"claims"`
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
	w.claims = make(map[BlockKey]string)
	for name, acc := range accounts {
		for _, c := range acc.Claims {
			w.claims[claimKey(c.X, c.Y, c.Z, c.Dim)] = name
		}
	}
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
			w.accounts[name] = &Account{Permissions: d.Permissions, Claims: []ClaimPos{}}
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
	w.accounts[name] = &Account{Claims: []ClaimPos{}}
	w.saveAccounts()
}

func (w *World) PermissionsOf(name string) Permissions {
	w.refreshPermissions()
	if acc, ok := w.accounts[name]; ok {
		return acc.Permissions
	}
	return Permissions{}
}

func claimKey(x, y, z, dim int32) BlockKey {
	return BlockKey{X: x, Y: byte(y), Z: z, Dim: dim}
}

func (w *World) ClaimOwner(x, y, z, dim int32) (string, bool) {
	if y < 0 || y >= CHUNK_SIZE_Y {
		return "", false
	}
	owner, ok := w.claims[claimKey(x, y, z, dim)]
	return owner, ok
}

func (w *World) IsClaimed(x, y, z, dim int32) bool {
	_, ok := w.ClaimOwner(x, y, z, dim)
	return ok
}

func (w *World) TouchesClaim(x, y, z, dim int32) bool {
	for _, n := range neighbours {
		if w.IsClaimed(x+n.Dx, y+n.Dy, z+n.Dz, dim) {
			return true
		}
	}
	return false
}

func (w *World) Claim(name string, x, y, z, dim int32) {
	acc := w.accounts[name]
	acc.Claims = append(acc.Claims, ClaimPos{X: x, Y: y, Z: z, Dim: dim})
	w.claims[claimKey(x, y, z, dim)] = name
	w.saveAccounts()
}

func (w *World) Unclaim(x, y, z, dim int32) {
	owner := w.claims[claimKey(x, y, z, dim)]
	delete(w.claims, claimKey(x, y, z, dim))
	acc := w.accounts[owner]
	for i, c := range acc.Claims {
		if c == (ClaimPos{X: x, Y: y, Z: z, Dim: dim}) {
			acc.Claims = append(acc.Claims[:i], acc.Claims[i+1:]...)
			break
		}
	}
	w.saveAccounts()
}
