package packethandler

import (
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/player"
)

var betacraftProxy, _ = url.Parse("http://betacraft.ee:11705")

var authClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		Proxy:             http.ProxyURL(betacraftProxy),
		DisableKeepAlives: true,
	},
}

var mojangClient = &http.Client{Timeout: 5 * time.Second}

func isPremium(username string) bool {
	resp, err := mojangClient.Get("https://api.mojang.com/users/profiles/minecraft/" + url.PathEscape(username))
	if err != nil {
		log.Printf("Failed to look up %s, requiring auth: %v", username, err)
		return true
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return true
	case http.StatusNotFound, http.StatusNoContent:
		return false
	}
	log.Printf("Failed to look up %s, requiring auth: status %d", username, resp.StatusCode)
	return true
}

func requiresAuth(world *level.World, username string) bool {
	if world.OnlineMode || !offlineName.MatchString(username) {
		return true
	}
	claimed := make(chan bool, 1)
	world.Enqueue(func() { claimed <- !world.AllowsOffline(username) })
	return <-claimed || isPremium(username)
}

func newServerId() string {
	return strconv.FormatUint(rand.Uint64(), 16)
}

func verifyUsername(username, serverId string) bool {
	q := url.Values{"user": {username}, "serverId": {serverId}}
	resp, err := authClient.Get("http://www.minecraft.net/game/checkserver.jsp?" + q.Encode())
	if err != nil {
		log.Printf("Failed to verify %s: %v", username, err)
		return false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		log.Printf("Failed to verify %s: %v", username, err)
		return false
	}
	return strings.TrimSpace(string(body)) == "YES"
}

var offlineName = regexp.MustCompile(`^[A-Za-z0-9_]{1,16}$`)

func sendAuthPrompt(world *level.World, pl *player.Player) {
	if world.PasswordOf(pl.Username) == "" {
		sendDebugMessage(pl, "Please register with /register <password>")
	} else {
		sendDebugMessage(pl, "Please log in with /login <password>")
	}
}

func pendingSession(world *level.World, pl *player.Player) bool {
	cur, ok := world.GetPlayerByUsername(pl.Username)
	return ok && cur == pl && pl.AuthPending
}

func startAuthTimeout(world *level.World, pl *player.Player) {
	time.AfterFunc(60*time.Second, func() {
		world.Enqueue(func() {
			if !pendingSession(world, pl) {
				return
			}
			log.Printf("%s did not log in within 60s", pl.Username)
			kickConnection(pl.Connection, "Login timed out")
		})
	})
}

func HoldPendingAuth(world *level.World) {
	world.ForEachPlayer(func(pl *player.Player) {
		if pl.AuthPending {
			sendPlayerPositionAndLook(pl.Connection, pl.X, pl.Z, pl.Y)
		}
	})
}

func handleAuthCommand(pl *player.Player, world *level.World, message string) {
	args := strings.Fields(message)
	stored := world.PasswordOf(pl.Username)
	current := func() bool { return pendingSession(world, pl) }
	switch {
	case len(args) == 2 && args[0] == "/register" && stored == "":
		go func() {
			hash := level.HashPassword(args[1])
			world.Enqueue(func() {
				if !current() || world.PasswordOf(pl.Username) != "" {
					return
				}
				world.RegisterOffline(pl.Username, hash)
				log.Printf("%s registered an offline account", pl.Username)
				sendAuthPrompt(world, pl)
			})
		}()
	case len(args) == 2 && args[0] == "/login" && stored != "":
		go func() {
			ok := level.VerifyPassword(stored, args[1])
			world.Enqueue(func() {
				if !current() {
					return
				}
				if !ok {
					log.Printf("Wrong password for offline account %s from %s", pl.Username, pl.Connection.RemoteAddr())
					kickConnection(pl.Connection, "Wrong password")
					return
				}
				pl.AuthPending = false
				world.RecordIP(pl.Username, remoteIP(pl.Connection))
				sendDebugMessage(pl, "Logged in.")
			})
		}()
	default:
		sendAuthPrompt(world, pl)
	}
}
