package level

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func (w *World) LoadBans() error {
	var err error
	if w.bannedPlayers, err = readBanList(filepath.Join(w.WorldDir, "banned_players.txt")); err != nil {
		return err
	}
	w.bannedIPs, err = readBanList(filepath.Join(w.WorldDir, "banned_ips.txt"))
	return err
}

func readBanList(path string) (map[string]bool, error) {
	set := make(map[string]bool)
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, entry := range strings.Fields(string(raw)) {
		set[strings.ToLower(entry)] = true
	}
	return set, nil
}

func (w *World) addBan(set map[string]bool, file, entry string) {
	if entry == "" || set[entry] {
		return
	}
	set[entry] = true
	f, err := os.OpenFile(filepath.Join(w.WorldDir, file), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Println("Failed to write", file, err)
		return
	}
	defer f.Close()
	fmt.Fprintln(f, entry)
}

func (w *World) IsBanned(name, ip string) bool {
	return w.bannedPlayers[strings.ToLower(name)] || w.bannedIPs[ip]
}

func (w *World) Ban(name, ip string) {
	w.addBan(w.bannedPlayers, "banned_players.txt", strings.ToLower(name))
	w.addBan(w.bannedIPs, "banned_ips.txt", ip)
}
