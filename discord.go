package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leNicDev/retromc/level"
	"github.com/leNicDev/retromc/packet/packets"
)

const (
	discordWebhookName = "retromc"
	discordLogFlush    = 2 * time.Second
	discordMaxMsgLen   = 1900
	mcChatLineLen      = 100
	mcMaxDiscordLines  = 3
)

const mcAllowedChars = " !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_abcdefghijklmnopqrstuvwxyz{|}~⌂ÇüéâäàåçêëèïîìÄÅÉæÆôöòûùÿÖÜø£Ø×ƒáíóúñÑªº¿®¬½¼¡«»"

var (
	discordStderr      = log.New(os.Stderr, "discord: ", log.LstdFlags)
	discordEmojiRe     = regexp.MustCompile(`<a?(:\w+:)\d+>`)
	discordMarkdownEsc = strings.NewReplacer(`\`, `\\`, `*`, `\*`, `_`, `\_`, `~`, `\~`, "`", "\\`", `|`, `\|`, `>`, `\>`)
)

var activeDiscord atomic.Pointer[discordBridge]

var discordNoMentions = &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}

type discordMsg struct {
	name  string
	text  string
	event bool
}

type discordReq struct {
	url     string
	bot     bool
	payload any
}

type discordBridge struct {
	session      *discordgo.Session
	world        *level.World
	http         *http.Client
	chatChannel  string
	logChannel   string
	webhookID    string
	webhookToken string
	outbox       chan discordMsg
	logs         chan string
	dropped      atomic.Int64
	nextSend     atomic.Int64
}

type discordLogWriter struct {
	logs chan string
}

func (w discordLogWriter) Write(p []byte) (int, error) {
	select {
	case w.logs <- string(p):
	default:
	}
	return len(p), nil
}

func startDiscord(world *level.World) {
	token := os.Getenv("DISCORD_TOKEN")
	chatChannel := os.Getenv("DISCORD_CHANNEL_ID")
	if token == "" || chatChannel == "" {
		log.Println("Discord not configured (DISCORD_TOKEN/DISCORD_CHANNEL_ID); integration disabled")
		return
	}

	discordgo.Logger = func(msgL, caller int, format string, a ...interface{}) {
		discordStderr.Printf(format, a...)
	}

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Println("Discord: failed to create session:", err)
		return
	}
	s.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentMessageContent

	b := &discordBridge{
		session:     s,
		world:       world,
		http:        &http.Client{Timeout: 10 * time.Second},
		chatChannel: chatChannel,
		logChannel:  os.Getenv("DISCORD_LOG_CHANNEL_ID"),
		outbox:      make(chan discordMsg, 1000),
		logs:        make(chan string, 1024),
	}

	if b.logChannel != "" {
		log.SetOutput(io.MultiWriter(os.Stderr, discordLogWriter{logs: b.logs}))
	}

	s.AddHandler(b.onMessage)

	go func() {
		backoff := time.Minute
		for {
			err := s.Open()
			if err == nil {
				break
			}
			discordStderr.Printf("failed to connect (likely a Cloudflare IP ban on a shared host), retrying in %v: %v", backoff, err)
			time.Sleep(backoff)
			backoff = min(backoff*2, 30*time.Minute)
		}
		b.setupWebhook()
		world.Enqueue(func() { world.SetChatRelay(b.relay) })
		activeDiscord.Store(b)
		b.outbox <- discordMsg{text: "Server is online", event: true}
		go b.runSender()
		log.Println("Discord bridge connected")
	}()
}

func announceDiscordShutdown() {
	b := activeDiscord.Load()
	if b == nil {
		return
	}
	done := make(chan struct{})
	go func() {
		if _, _, err := b.post(b.statusReq("Server is shutting down", 0xFF5555)); err != nil {
			discordStderr.Println("failed to send shutdown message:", err)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
}

func (b *discordBridge) setupWebhook() {
	hooks, err := b.session.ChannelWebhooks(b.chatChannel)
	if err == nil {
		for _, h := range hooks {
			if h.Name == discordWebhookName && h.Token != "" {
				b.webhookID, b.webhookToken = h.ID, h.Token
				return
			}
		}
	}
	h, err := b.session.WebhookCreate(b.chatChannel, discordWebhookName, "")
	if err != nil {
		discordStderr.Println("cannot create webhook (needs Manage Webhooks), falling back to plain bot messages:", err)
		return
	}
	b.webhookID, b.webhookToken = h.ID, h.Token
}

func (b *discordBridge) relay(name, text string, event bool) {
	select {
	case b.outbox <- discordMsg{name: name, text: text, event: event}:
	default:
		b.dropped.Add(1)
	}
}

func (b *discordBridge) statusReq(text string, color int) discordReq {
	return discordReq{url: discordgo.EndpointChannelMessages(b.chatChannel), bot: true, payload: &discordgo.MessageSend{
		Embeds:          []*discordgo.MessageEmbed{{Description: "**" + text + "**", Color: color}},
		AllowedMentions: discordNoMentions,
	}}
}

func (b *discordBridge) chatReq(m discordMsg) discordReq {
	if m.name == "" {
		return b.statusReq(m.text, 0x55FF55)
	}
	avatar := "https://mc-heads.net/avatar/" + url.PathEscape(m.name) + "/64"
	switch {
	case m.event:
		return discordReq{url: discordgo.EndpointChannelMessages(b.chatChannel), bot: true, payload: &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{{
				Author: &discordgo.MessageEmbedAuthor{Name: m.text, IconURL: avatar},
				Color:  0xFFFF55,
			}},
			AllowedMentions: discordNoMentions,
		}}
	case b.webhookID != "":
		return discordReq{url: discordgo.EndpointWebhookToken(b.webhookID, b.webhookToken), payload: &discordgo.WebhookParams{
			Content:         discordMarkdownEsc.Replace(m.text),
			Username:        m.name,
			AvatarURL:       avatar,
			AllowedMentions: discordNoMentions,
		}}
	default:
		return discordReq{url: discordgo.EndpointChannelMessages(b.chatChannel), bot: true, payload: &discordgo.MessageSend{
			Content:         fmt.Sprintf("**%s**: %s", discordMarkdownEsc.Replace(m.name), discordMarkdownEsc.Replace(m.text)),
			AllowedMentions: discordNoMentions,
		}}
	}
}

func (b *discordBridge) runSender() {
	ticker := time.NewTicker(discordLogFlush)
	defer ticker.Stop()
	var logBuf strings.Builder
	skippedLogs := 0
	for {
		select {
		case m := <-b.outbox:
			b.deliver(b.chatReq(m))
			if n := b.dropped.Swap(0); n > 0 {
				b.deliver(b.statusReq(fmt.Sprintf("%d message(s) could not be relayed to Discord", n), 0xFF5555))
			}
		case line := <-b.logs:
			if b.logChannel == "" {
				continue
			}
			line = truncateRunes(strings.ReplaceAll(line, "```", "'''"), discordMaxMsgLen/4)
			if !strings.HasSuffix(line, "\n") {
				line += "\n"
			}
			if logBuf.Len()+len(line) > discordMaxMsgLen {
				skippedLogs++
				continue
			}
			logBuf.WriteString(line)
		case <-ticker.C:
			if logBuf.Len() == 0 && skippedLogs == 0 {
				continue
			}
			content := "```\n" + logBuf.String() + "```"
			if skippedLogs > 0 {
				content += fmt.Sprintf("(%d log line(s) skipped)", skippedLogs)
			}
			b.deliver(discordReq{url: discordgo.EndpointChannelMessages(b.logChannel), bot: true, payload: &discordgo.MessageSend{
				Content:         content,
				AllowedMentions: discordNoMentions,
			}})
			logBuf.Reset()
			skippedLogs = 0
		}
	}
}

func (b *discordBridge) deliver(req discordReq) {
	backoff := 5 * time.Second
	for {
		if wait := time.Until(time.Unix(0, b.nextSend.Load())); wait > 0 {
			time.Sleep(wait)
		}
		retry, wait, err := b.post(req)
		if err == nil {
			return
		}
		if !retry {
			discordStderr.Println("dropping message:", err)
			return
		}
		if wait <= 0 {
			wait = backoff
			backoff = min(backoff*2, 5*time.Minute)
		}
		discordStderr.Printf("send failed, retrying in %v: %v", wait, err)
		time.Sleep(wait)
	}
}

func (b *discordBridge) post(req discordReq) (retry bool, wait time.Duration, err error) {
	body, err := json.Marshal(req.payload)
	if err != nil {
		return false, 0, err
	}
	hr, err := http.NewRequest(http.MethodPost, req.url, bytes.NewReader(body))
	if err != nil {
		return false, 0, err
	}
	hr.Header.Set("Content-Type", "application/json")
	hr.Header.Set("User-Agent", b.session.UserAgent)
	if req.bot {
		hr.Header.Set("Authorization", b.session.Token)
	}

	resp, err := b.http.Do(hr)
	if err != nil {
		return true, 0, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))

	if resp.Header.Get("X-RateLimit-Remaining") == "0" {
		if after, perr := strconv.ParseFloat(resp.Header.Get("X-RateLimit-Reset-After"), 64); perr == nil {
			b.nextSend.Store(time.Now().Add(time.Duration(after * float64(time.Second))).UnixNano())
		}
	}

	switch {
	case resp.StatusCode < 300:
		return false, 0, nil
	case resp.StatusCode == http.StatusTooManyRequests:
		var rl struct {
			RetryAfter float64 `json:"retry_after"`
		}
		if json.Unmarshal(respBody, &rl) == nil && rl.RetryAfter > 0 {
			return true, time.Duration(rl.RetryAfter*float64(time.Second)) + 250*time.Millisecond, fmt.Errorf("rate limited: %s", respBody)
		}
		return true, 0, fmt.Errorf("blocked by Cloudflare (shared IP ban?): %s", truncateRunes(string(respBody), 200))
	case resp.StatusCode >= 500 || !bytes.HasPrefix(bytes.TrimSpace(respBody), []byte("{")):
		return true, 0, fmt.Errorf("discord %s: %s", resp.Status, truncateRunes(string(respBody), 200))
	default:
		return false, 0, fmt.Errorf("discord %s: %s", resp.Status, truncateRunes(string(respBody), 300))
	}
}

func (b *discordBridge) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.ChannelID != b.chatChannel || m.Author == nil || m.Author.Bot || m.WebhookID != "" {
		return
	}

	name := m.Author.GlobalName
	if m.Member != nil && m.Member.Nick != "" {
		name = m.Member.Nick
	}
	if name == "" {
		name = m.Author.Username
	}
	name = truncateRunes(sanitizeForMC(name), 16)
	if name == "" {
		name = "?"
	}

	text := discordEmojiRe.ReplaceAllString(m.ContentWithMentionsReplaced(), "$1")
	text = strings.Join(strings.Fields(sanitizeForMC(text)), " ")
	if len(m.Attachments) > 0 {
		text = strings.TrimSpace(text + " [attachment]")
	}
	if text == "" {
		return
	}

	prefix := "§9<" + name + ">§f "
	lines := wrapRunes(text, mcChatLineLen-len([]rune(prefix)))
	if len(lines) > mcMaxDiscordLines {
		lines = lines[:mcMaxDiscordLines]
		lines[len(lines)-1] = truncateRunes(lines[len(lines)-1], mcChatLineLen-len([]rune(prefix))-3) + "..."
	}

	b.world.Enqueue(func() {
		for i, line := range lines {
			if i == 0 {
				line = prefix + line
			} else {
				line = "§f" + line
			}
			p := packets.ChatMessagePacket{Message: line}
			b.world.BroadcastPacket(p.Serialize())
		}
	})
}

func sanitizeForMC(s string) string {
	var out strings.Builder
	for _, r := range s {
		if r == '`' {
			r = '\''
		}
		if strings.ContainsRune(mcAllowedChars, r) {
			out.WriteRune(r)
		}
	}
	return strings.TrimSpace(out.String())
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

func wrapRunes(s string, n int) []string {
	r := []rune(s)
	var lines []string
	for len(r) > n {
		cut := n
		for i := n; i > n/2; i-- {
			if r[i] == ' ' {
				cut = i
				break
			}
		}
		lines = append(lines, strings.TrimSpace(string(r[:cut])))
		r = r[cut:]
	}
	return append(lines, strings.TrimSpace(string(r)))
}
