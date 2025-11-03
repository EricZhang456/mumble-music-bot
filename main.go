package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/EricZhang456/mumble-music-bot/bot"
	"github.com/EricZhang456/mumble-music-bot/media"
	"github.com/joho/godotenv"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	godotenv.Load()

	dbPath := os.Getenv("BOT_DB_PATH")
	if dbPath == "" {
		log.Fatal("DB path is empty.")
	}

	musicPath := os.Getenv("MUSIC_PATH")
	if musicPath == "" {
		log.Fatal("Music path is empty.")
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to open database: ", err)
	}

	if err := db.AutoMigrate(&media.AudioData{}); err != nil {
		log.Fatal("Failed to migrate database: ", err)
	}

	scanner := media.CreateAudioScanner(db)
	log.Println("Scanning audio files.")
	if err := scanner.ScanAndWriteToDb(musicPath); err != nil {
		log.Fatal("Failed to scan audio files: ", err)
	}

	botUsername := strings.TrimSpace(os.Getenv("MUMBLE_USER"))
	botCommandPrefix := strings.TrimSpace(os.Getenv("COMMAND_PREFIX"))
	botPassword := strings.TrimSpace(os.Getenv("MUMBLE_PASSWORD"))

	if botUsername == "" || botCommandPrefix == "" {
		log.Fatal("Bot username and command prefix cannot be empty.")
	}

	options := []bot.MumbleOptions{}
	if botPassword != "" {
		options = append(options, bot.WithPassword(botPassword))
	}

	mb := bot.CreateMumbleBot(botUsername, options...)
	mumbleServer := os.Getenv("MUMBLE_SERVER")
	mumblePortEnv := os.Getenv("MUMBLE_PORT")
	var mumblePort int
	if mumblePortEnv != "" {
		var err error
		mumblePort, err = strconv.Atoi(mumblePortEnv)
		if err != nil {
			log.Fatal("Invalid mumble port.")
		}
	} else {
		mumblePort = 64738
	}

	log.Println("Joining Mumble server.")
	mb.Connect(mumbleServer, mumblePort)
	mumbleChannel := os.Getenv("MUMBLE_CHANNEL")
	if mumbleChannel != "" {
		mb.JoinChannel(mumbleChannel)
	}

	player := bot.CreateMusicPlayer(mb)
	commandHandler := bot.CreateCommandHandler(botCommandPrefix, player, db)
	mb.SetCommandHandler(commandHandler)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}
