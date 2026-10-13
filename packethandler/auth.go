package packethandler

import (
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var betacraftProxy, _ = url.Parse("http://betacraft.ee:11705")

var authClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		Proxy:             http.ProxyURL(betacraftProxy),
		DisableKeepAlives: true,
	},
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
