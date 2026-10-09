package main

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"regexp"
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

type discordMsg struct {
	name  string
	text  string
	event bool
}

type discordBridge struct {
	session      *discordgo.Session
	world        *level.World
	chatChannel  string
	logChannel   string
	webhookID    string
	webhookToken string
	outbox       chan discordMsg
	logs         chan string
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
		chatChannel: chatChannel,
		logChannel:  os.Getenv("DISCORD_LOG_CHANNEL_ID"),
		outbox:      make(chan discordMsg, 256),
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
		go b.runOutbox()
		if b.logChannel != "" {
			go b.runLogs()
		}
		activeDiscord.Store(b)
		b.sendStatus("Server is online", 0x55FF55)
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
		b.sendStatus("Server is shutting down", 0xFF5555)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
}

func (b *discordBridge) sendStatus(text string, color int) {
	_, err := b.session.ChannelMessageSendEmbed(b.chatChannel, &discordgo.MessageEmbed{
		Description: "**" + text + "**",
		Color:       color,
	})
	if err != nil {
		discordStderr.Println("failed to send status message:", err)
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
	}
}

func (b *discordBridge) runOutbox() {
	noMentions := &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}}
	for m := range b.outbox {
		avatar := "https://mc-heads.net/avatar/" + url.PathEscape(m.name) + "/64"
		var err error
		switch {
		case m.event:
			_, err = b.session.ChannelMessageSendComplex(b.chatChannel, &discordgo.MessageSend{
				Embeds: []*discordgo.MessageEmbed{{
					Author: &discordgo.MessageEmbedAuthor{Name: m.text, IconURL: avatar},
					Color:  0xFFFF55,
				}},
				AllowedMentions: noMentions,
			})
		case b.webhookID != "":
			_, err = b.session.WebhookExecute(b.webhookID, b.webhookToken, false, &discordgo.WebhookParams{
				Content:         discordMarkdownEsc.Replace(m.text),
				Username:        m.name,
				AvatarURL:       avatar,
				AllowedMentions: noMentions,
			})
		default:
			_, err = b.session.ChannelMessageSendComplex(b.chatChannel, &discordgo.MessageSend{
				Content:         fmt.Sprintf("**%s**: %s", discordMarkdownEsc.Replace(m.name), discordMarkdownEsc.Replace(m.text)),
				AllowedMentions: noMentions,
			})
		}
		if err != nil {
			discordStderr.Println("failed to send chat message:", err)
		}
	}
}

func (b *discordBridge) runLogs() {
	ticker := time.NewTicker(discordLogFlush)
	defer ticker.Stop()
	var buf strings.Builder
	flush := func() {
		if buf.Len() == 0 {
			return
		}
		_, err := b.session.ChannelMessageSendComplex(b.logChannel, &discordgo.MessageSend{
			Content:         "```\n" + buf.String() + "```",
			AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}},
		})
		if err != nil {
			discordStderr.Println("failed to send logs:", err)
		}
		buf.Reset()
	}
	for {
		select {
		case line := <-b.logs:
			line = strings.ReplaceAll(line, "```", "'''")
			if len(line) > discordMaxMsgLen {
				line = truncateRunes(line, discordMaxMsgLen/4) + "\n"
			}
			if buf.Len()+len(line) > discordMaxMsgLen {
				flush()
			}
			buf.WriteString(line)
		case <-ticker.C:
			flush()
		}
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
