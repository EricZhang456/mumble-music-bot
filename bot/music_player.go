package bot

import (
	"sync"

	"github.com/EricZhang456/mumble-music-bot/media"
	"github.com/EricZhang456/mumble-music-bot/utils"
)

type PlaybackMode int

const (
	Single PlaybackMode = iota
	Shuffle
	ShuffleRepeat
	Repeat
)

type MusicPlayer struct {
	bot          *MumbleBot
	playlist     []*media.AudioData
	mode         PlaybackMode
	currentIndex int
	mu           sync.Mutex
}

func CreateMusicPlayer(bot *MumbleBot) *MusicPlayer {
	musicPlayer := &MusicPlayer{bot: bot}
	return musicPlayer
}

func (mp *MusicPlayer) AddToPlaylist(track media.AudioData) {
	mp.mu.Lock()
	mp.playlist = append(mp.playlist, &track)
	mp.mu.Unlock()
}

func (mp *MusicPlayer) AddAllToPlaylist(tracks []media.AudioData) {
	utils.ReverseList(tracks)
	for _, track := range tracks {
		mp.AddToPlaylist(track)
	}
}

func (mp *MusicPlayer) SetMode(mode PlaybackMode) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.mode = mode
	if mode == Shuffle {
		utils.ShuffleList(mp.playlist)
		mp.currentIndex = 0
	}
}

func (mp *MusicPlayer) StartPlaylist() {
	mp.mu.Lock()

	if len(mp.playlist) == 0 {
		mp.mu.Unlock()
		return
	}

	if mp.currentIndex >= len(mp.playlist) {
		switch mp.mode {
		case ShuffleRepeat:
			utils.ShuffleList(mp.playlist)
			fallthrough
		case Repeat:
			mp.currentIndex = 0
		default:
			mp.mu.Unlock()
			mp.StopPlaylist()
			return
		}
	}

	track := mp.playlist[mp.currentIndex]
	mp.mu.Unlock()

	mp.bot.PlayAudio(track, func() {
		mp.mu.Lock()
		defer mp.mu.Unlock()
		mp.currentIndex++

		if mp.mode == Single || mp.mode == Shuffle {
			if mp.currentIndex >= len(mp.playlist) {
				return
			}
		}

		mp.StartPlaylist()
	})
}

func (mp *MusicPlayer) StopPlaylist() {
	mp.mu.Lock()
	mp.playlist = nil
	mp.currentIndex = 0
	mp.mu.Unlock()
	mp.bot.StopAudio()
}
