package packethandler

import (
	"net"
	"testing"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/player"
)

func banWorld(t *testing.T, dir string) *level.World {
	w := level.NewWorld("test", 1, level.Default)
	w.WorldDir = dir
	if err := w.LoadAccounts(); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadBans(); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestBanUnban(t *testing.T) {
	dir := t.TempDir()
	w := banWorld(t, dir)
	w.RegisterOffline("Steve", "x")
	w.RecordIP("Steve", "1.2.3.4")

	t.Log(BanCommand(w, "test", []string{"steve"}))
	if !w.IsBanned("Steve", "9.9.9.9") || !w.IsBanned("Alt", "1.2.3.4") {
		t.Fatal("ban did not cover name and IP")
	}
	w = banWorld(t, dir)
	if !w.IsBanned("STEVE", "") || !w.IsBanned("", "1.2.3.4") {
		t.Fatal("ban not persisted")
	}

	t.Log(UnbanCommand(w, "test", []string{"Steve"}))
	if w.IsBanned("Steve", "1.2.3.4") {
		t.Fatal("unban did not lift name and IP")
	}
	w = banWorld(t, dir)
	if w.IsBanned("Steve", "1.2.3.4") {
		t.Fatal("unban not persisted")
	}
	if got := UnbanCommand(w, "test", []string{"Steve"}); got[0] != "Steve was not banned" {
		t.Fatal(got)
	}

	BanCommand(w, "test", []string{"Steve"})
	t.Log(UnbanCommand(w, "test", []string{"1.2.3.4"}))
	if !w.IsBanned("Steve", "") || w.IsBanned("Alt", "1.2.3.4") {
		t.Fatal("IP unban should lift only the IP")
	}
}

func TestBanOnlinePlayer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	server, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	w := banWorld(t, dir)
	w.RegisterOffline("Herb", "x")
	pl := &player.Player{Username: "Herb", Connection: server, LoggedIn: true}
	w.AddPlayer(pl)

	t.Log(BanCommand(w, "test", []string{"herb"}))
	if !w.IsBanned("Other", "127.0.0.1") {
		t.Fatal("live connection IP was not banned")
	}
	buf := make([]byte, 1)
	if _, err := client.Read(buf); err != nil || buf[0] != 0xFF {
		t.Fatalf("expected kick packet, got %x %v", buf, err)
	}
	t.Log(DreamCommand(w, []string{"nobody"}))

	w = banWorld(t, dir)
	t.Log(UnbanCommand(w, "test", []string{"Herb"}))
	if w.IsBanned("Herb", "127.0.0.1") {
		t.Fatal("unban after restart missed the live IP")
	}
}
