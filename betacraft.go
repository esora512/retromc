package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/leNicDev/retromc/level"
)

const (
	betacraftUpdateURL  = "https://api.betacraft.uk/v2/server_update"
	betacraftMaxPlayers = 20
)

type betacraftUpdate struct {
	GameVersion   string            `json:"game_version"`
	Protocol      string            `json:"protocol"`
	V1Version     string            `json:"v1_version"`
	Category      string            `json:"category"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	PrivateKey    string            `json:"private_key"`
	Socket        string            `json:"socket"`
	SendPlayers   bool              `json:"send_players"`
	MaxPlayers    int               `json:"max_players"`
	OnlinePlayers int               `json:"online_players"`
	OnlineMode    bool              `json:"online_mode"`
	Software      map[string]string `json:"software"`
	Players       []any             `json:"players"`
}

func startBetacraftHeartbeat(world *level.World) {
	socket := os.Getenv("BETACRAFT_SOCKET")
	if socket == "" {
		log.Println("Betacraft heartbeat not configured (BETACRAFT_SOCKET); server list disabled")
		return
	}
	name := os.Getenv("BETACRAFT_NAME")
	if name == "" {
		name = "retromc"
	}
	update := betacraftUpdate{
		GameVersion: "b1.7.3",
		Protocol:    "beta_14",
		V1Version:   "b1.7.3",
		Category:    "beta",
		Name:        name,
		Description: os.Getenv("BETACRAFT_DESCRIPTION"),
		PrivateKey:  os.Getenv("BETACRAFT_KEY"),
		Socket:      socket,
		SendPlayers: false,
		Players:     []any{},
		MaxPlayers:  betacraftMaxPlayers,
		OnlineMode:  world.OnlineMode,
		Software:    map[string]string{"name": "retromc", "version": GitCommit},
	}
	client := &http.Client{Timeout: 10 * time.Second}

	go func() {
		lastStatus := ""
		for {
			update.OnlinePlayers = onlineCount(world)
			status := "Betacraft heartbeat OK for " + socket
			if err := sendBetacraftUpdate(client, update); err != nil {
				status = "Betacraft heartbeat failed: " + err.Error()
			}
			if status != lastStatus {
				log.Println(status)
				lastStatus = status
			}
			time.Sleep(time.Minute)
		}
	}()
}

func onlineCount(world *level.World) int {
	done := make(chan int, 1)
	world.Enqueue(func() { done <- world.OnlineCount() })
	return <-done
}

func sendBetacraftUpdate(client *http.Client, update betacraftUpdate) error {
	body, err := json.Marshal(update)
	if err != nil {
		return err
	}
	resp, err := client.Post(betacraftUpdateURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var result struct {
		Error bool `json:"error"`
	}
	if json.Unmarshal(raw, &result) != nil || result.Error || resp.StatusCode >= 400 {
		return fmt.Errorf("%s: %s", resp.Status, raw)
	}
	return nil
}
