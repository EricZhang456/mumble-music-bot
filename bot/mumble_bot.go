package bot

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/EricZhang456/mumble-music-bot/media"
	"layeh.com/gumble/gumble"
	"layeh.com/gumble/gumbleffmpeg"
	"layeh.com/gumble/gumbleutil"
	_ "layeh.com/gumble/opus"
)

type MumbleBot struct {
	command          string
	client           *gumble.Client
	config           *gumble.Config
	currentAudioData *media.AudioData
	currentStream    *gumbleffmpeg.Stream
	mu               sync.Mutex
}

type MumbleOptions func(*gumble.Config)

func WithPassword(password string) MumbleOptions {
	return func(c *gumble.Config) {
		c.Password = password
	}
}

func WithTokens(tokens []string) MumbleOptions {
	return func(c *gumble.Config) {
		c.Tokens = tokens
	}
}

func CreateMumbleBot(username string, command string, opts ...MumbleOptions) *MumbleBot {
	cfg := gumble.NewConfig()
	cfg.Username = username
	for _, opt := range opts {
		opt(cfg)
	}
	bot := &MumbleBot{config: cfg, command: command}
	cfg.Attach(gumbleutil.Listener{
		TextMessage: bot.onTextMessage,
	})
	return bot
}

func (bot *MumbleBot) Connect(host string, port int) {
	targetHost := host + ":" + strconv.Itoa(port)
	var err error
	bot.client, err = gumble.Dial(targetHost, bot.config)
	if err != nil {
		panic(err)
	}
}

func (bot *MumbleBot) JoinChannel(channel string) {
	ch := bot.client.Channels.Find(channel)
	if ch == nil {
		return
	}
	bot.client.Self.Move(ch)
}

func (bot *MumbleBot) onTextMessage(e *gumble.TextMessageEvent) {
	if e.Sender == nil {
		return
	}
	ch := bot.client.Self.Channel
	if ch == nil {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(e.Message), bot.command) {
		return
	}
	if bot.currentAudioData == nil {
		ch.Send("Not playing anything right now.", false)
		return
	}
	nowPlaying := bot.currentAudioData.Title
	if bot.currentAudioData.Artists != nil && len(strings.TrimSpace(*bot.currentAudioData.Artists)) > 0 {
		nowPlaying = *bot.currentAudioData.Artists + " - " + nowPlaying
	}
	if bot.currentAudioData.Album != nil && len(strings.TrimSpace(*bot.currentAudioData.Album)) > 0 {
		nowPlaying += " (from " + *bot.currentAudioData.Album + ")"
	}
	ch.Send("<b>Now playing:</b> "+nowPlaying, false)
}

func (bot *MumbleBot) PlayAudio(data *media.AudioData, onComplete func()) {
	bot.mu.Lock()
	if bot.currentStream != nil {
		bot.mu.Unlock()
		return
	}
	bot.currentAudioData = data
	stream := gumbleffmpeg.New(bot.client, gumbleffmpeg.SourceFile(data.Path))
	bot.currentStream = stream
	bot.mu.Unlock()

	go func() {
		err := stream.Play()
		if err != nil {
			fmt.Printf("Playback error: %v\n", err)
		}
		stream.Wait()
		bot.mu.Lock()
		bot.currentStream = nil
		bot.currentAudioData = nil
		bot.mu.Unlock()

		if onComplete != nil {
			onComplete()
		}
	}()
}

func (bot *MumbleBot) StopAudio() {
	bot.mu.Lock()
	if bot.currentStream != nil {
		bot.currentStream.Stop()
		bot.currentStream = nil
		bot.currentAudioData = nil
	}
	bot.mu.Unlock()
}
