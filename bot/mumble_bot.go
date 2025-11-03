package bot

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/EricZhang456/mumble-music-bot/media"
	"layeh.com/gumble/gumble"
	"layeh.com/gumble/gumbleffmpeg"
	"layeh.com/gumble/gumbleutil"
	_ "layeh.com/gumble/opus"
)

type MumbleBot struct {
	client           *gumble.Client
	config           *gumble.Config
	currentAudioData *media.AudioData
	currentStream    *gumbleffmpeg.Stream
	commandHandler   CommandHandler
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

func CreateMumbleBot(username string, opts ...MumbleOptions) *MumbleBot {
	cfg := gumble.NewConfig()
	cfg.Username = username
	for _, opt := range opts {
		opt(cfg)
	}
	bot := &MumbleBot{config: cfg}
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

func (bot *MumbleBot) SetCommandHandler(commandHandler CommandHandler) {
	bot.mu.Lock()
	defer bot.mu.Unlock()
	bot.commandHandler = commandHandler
}

func (bot *MumbleBot) onTextMessage(e *gumble.TextMessageEvent) {
	if e.Sender == nil {
		return
	}
	ch := bot.client.Self.Channel
	if ch == nil {
		return
	}
	if bot.commandHandler == nil {
		ch.Send("No command handler has been registered yet.", false)
	}
	message := bot.commandHandler.HandleCommand(e.Message)
	if message != nil {
		ch.Send(*message, false)
	}
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
