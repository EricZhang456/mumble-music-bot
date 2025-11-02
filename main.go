package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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

	botUsername := os.Getenv("MUMBLE_USER")
	botCommand := os.Getenv("NOWPLAYING_COMMAND")
	botPassword := os.Getenv("MUMBLE_PASSWORD")

	options := []bot.MumbleOptions{}
	if botPassword != "" {
		options = append(options, bot.WithPassword(botPassword))
	}

	mb := bot.CreateMumbleBot(botUsername, botCommand, options...)
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

	mb.Connect(mumbleServer, mumblePort)
	mumbleChannel := os.Getenv("MUMBLE_CHANNEL")
	if mumbleChannel != "" {
		mb.JoinChannel(mumbleChannel)
	}

	time.Sleep(5 * time.Second)
	mb.JoinChannel(mumbleChannel)

	player := bot.CreateMusicPlayer(mb)
	endpoint := bot.CreateMusicPlayerEndpoint(player, db)

	http.HandleFunc("GET /tracks", endpoint.GetAllAudioDataHandler)
	http.HandleFunc("POST /add_single", endpoint.AddSingleTrackHandler)
	http.HandleFunc("POST /add_all", endpoint.AddAllTracksHandler)
	http.HandleFunc("POST /set_mode", endpoint.SetPlaybackModeHandler)
	http.HandleFunc("POST /start", endpoint.StartPlaylistHandler)
	http.HandleFunc("POST /stop", endpoint.StopPlaylistHandler)

	endpointPortStr := os.Getenv("CONTROLLER_ENDPOINT_PORT")
	var endpointPort int
	if endpointPortStr != "" {
		var err error
		endpointPort, err = strconv.Atoi(endpointPortStr)
		if err != nil {
			log.Fatal("CONTROLLER_ENDPOINT_PORT is not a valid port.")
		}
	} else {
		endpointPort = 8080
	}
	log.Println("Serving on :" + strconv.Itoa(endpointPort))
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(endpointPort), nil))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}
