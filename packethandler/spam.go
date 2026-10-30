package packethandler

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/player"
)

var chatLimits = []struct {
	window time.Duration
	max    int
}{
	{2 * time.Second, 1},
	{time.Minute, 8},
	{15 * time.Minute, 60},
}

const (
	strikeWindow   = 2 * time.Minute
	strikesToMute  = 3
	baseMute       = 5 * time.Minute
	maxMute        = 24 * time.Hour
	muteMemoryTime = 24 * time.Hour
)

type chatHistory struct {
	sent      []time.Time
	strikes   []time.Time
	mutes     int
	lastMuted time.Time
}

var (
	chatHistories = map[string]*chatHistory{}
	mutedUntil    = map[string]time.Time{}
)

func pruneBefore(ts []time.Time, cutoff time.Time) []time.Time {
	i := 0
	for i < len(ts) && ts[i].Before(cutoff) {
		i++
	}
	return ts[i:]
}

func muteKeys(pl *player.Player) []string {
	keys := []string{"name:" + strings.ToLower(pl.Username)}
	if ip := remoteIP(pl.Connection); ip != "" {
		keys = append(keys, "ip:"+ip)
	}
	return keys
}

func muteRemaining(pl *player.Player, now time.Time) time.Duration {
	var left time.Duration
	for _, k := range muteKeys(pl) {
		if until, ok := mutedUntil[k]; ok {
			if !now.Before(until) {
				delete(mutedUntil, k)
			} else if d := until.Sub(now); d > left {
				left = d
			}
		}
	}
	return left
}

func allowChat(pl *player.Player, world *level.World) bool {
	if world.PermissionsOf(pl.Username).CanBan {
		return true
	}
	now := time.Now()
	if left := muteRemaining(pl, now); left > 0 {
		sendDebugMessage(pl, fmt.Sprintf("\u00A7cYou are muted for %s.", left.Round(time.Second)))
		return false
	}
	if len(chatHistories) > 512 {
		sweepChatHistories(now)
	}
	key := strings.ToLower(pl.Username)
	h := chatHistories[key]
	if h == nil {
		h = &chatHistory{}
		chatHistories[key] = h
	}
	if h.mutes > 0 && now.Sub(h.lastMuted) > muteMemoryTime {
		h.mutes = 0
	}
	h.sent = pruneBefore(h.sent, now.Add(-chatLimits[len(chatLimits)-1].window))
	for _, l := range chatLimits {
		n := 0
		for _, t := range h.sent {
			if now.Sub(t) < l.window {
				n++
			}
		}
		if n >= l.max {
			return chatStrike(pl, h, now)
		}
	}
	h.sent = append(h.sent, now)
	return true
}

func chatStrike(pl *player.Player, h *chatHistory, now time.Time) bool {
	h.strikes = append(pruneBefore(h.strikes, now.Add(-strikeWindow)), now)
	if len(h.strikes) < strikesToMute {
		sendDebugMessage(pl, "\u00A7cYou are sending messages too fast.")
		return false
	}
	d := min(baseMute<<h.mutes, maxMute)
	h.mutes++
	h.lastMuted = now
	h.strikes = nil
	for _, k := range muteKeys(pl) {
		mutedUntil[k] = now.Add(d)
	}
	log.Printf("Muted %s (%s) for %s for spamming", pl.Username, remoteIP(pl.Connection), d)
	sendDebugMessage(pl, fmt.Sprintf("\u00A7cYou have been muted for %s for spamming.", d))
	return false
}

func sweepChatHistories(now time.Time) {
	for k, h := range chatHistories {
		if len(pruneBefore(h.sent, now.Add(-chatLimits[len(chatLimits)-1].window))) == 0 && now.Sub(h.lastMuted) > muteMemoryTime {
			delete(chatHistories, k)
		}
	}
	for k, until := range mutedUntil {
		if !now.Before(until) {
			delete(mutedUntil, k)
		}
	}
}
