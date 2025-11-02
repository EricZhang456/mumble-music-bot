package bot

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/EricZhang456/mumble-music-bot/media"
	"gorm.io/gorm"
)

type MusicData struct {
	Id       uint    `json:"id"`
	Title    string  `json:"title"`
	Artists  *string `json:"artists,omitempty"`
	Album    *string `json:"album,omitempty"`
	TrackNum *int    `json:"track_num,omitempty"`
	DiscNum  *int    `json:"disc_num,omitempty"`
}

type AddAllRequest struct {
	Ids []uint `json:"ids"`
}

type ModeResponse struct {
	Mode string `json:"mode"`
}

type MusicPlayerEndpoint struct {
	mp *MusicPlayer
	db *gorm.DB
}

func CreateMusicPlayerEndpoint(mp *MusicPlayer, db *gorm.DB) *MusicPlayerEndpoint {
	musicPlayerEndpoint := &MusicPlayerEndpoint{mp: mp, db: db}
	return musicPlayerEndpoint
}

func (e *MusicPlayerEndpoint) GetAllAudioDataHandler(w http.ResponseWriter, r *http.Request) {
	var audioData []media.AudioData
	if err := e.db.Find(&audioData).Error; err != nil {
		http.Error(w, "Audio query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	response := make([]MusicData, 0, len(audioData))
	for _, t := range audioData {
		response = append(response, MusicData{
			Id:       t.ID,
			Title:    t.Title,
			Artists:  t.Artists,
			Album:    t.Album,
			TrackNum: t.TrackNum,
			DiscNum:  t.DiscNum,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Cannot encode response: "+err.Error(), http.StatusInternalServerError)
	}
}

func (e *MusicPlayerEndpoint) getTrackById(id uint) (*media.AudioData, error) {
	var track media.AudioData
	if err := e.db.First(&track, id).Error; err != nil {
		return nil, err
	}
	return &track, nil
}

func (e *MusicPlayerEndpoint) AddSingleTrackHandler(w http.ResponseWriter, r *http.Request) {
	trackStr := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("trackid")))
	trackInt, err := strconv.Atoi(trackStr)
	if err != nil {
		http.Error(w, "Track id is not a number.", http.StatusBadRequest)
		return
	}
	track, err := e.getTrackById(uint(trackInt))
	if err != nil {
		http.Error(w, "Invalid track id.", http.StatusBadRequest)
		return
	}
	e.mp.AddToPlaylist(*track)
	w.WriteHeader(http.StatusOK)
}

func (e *MusicPlayerEndpoint) AddAllTracksHandler(w http.ResponseWriter, r *http.Request) {
	var req AddAllRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body.", http.StatusBadRequest)
		return
	}

	tracks := make([]media.AudioData, 0, len(req.Ids))
	for _, i := range req.Ids {
		track, err := e.getTrackById(i)
		if err != nil {
			http.Error(w, "Invalid track id.", http.StatusBadRequest)
			return
		}
		tracks = append(tracks, *track)
	}
	e.mp.AddAllToPlaylist(tracks)
	w.WriteHeader(http.StatusOK)
}

func (e *MusicPlayerEndpoint) SetPlaybackModeHandler(w http.ResponseWriter, r *http.Request) {
	modeStr := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("mode")))
	var mode PlaybackMode
	switch modeStr {
	case "single":
		mode = Single
	case "repeat":
		mode = Repeat
	case "shuffle":
		mode = Shuffle
	case "shufflerepeat":
		mode = ShuffleRepeat
	}
	e.mp.SetMode(mode)
	w.WriteHeader(http.StatusOK)
}

func (e *MusicPlayerEndpoint) GetPlaybackModeHandler(w http.ResponseWriter, r *http.Request) {
	var modeStr string
	switch e.mp.mode {
	case Single:
		modeStr = "single"
	case Repeat:
		modeStr = "repeat"
	case Shuffle:
		modeStr = "shuffle"
	case ShuffleRepeat:
		modeStr = "shufflerepeat"
	}
	response := ModeResponse{Mode: modeStr}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (e *MusicPlayerEndpoint) GetPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	response := make([]MusicData, 0, len(e.mp.playlist))
	for _, i := range e.mp.playlist {
		response = append(response, MusicData{
			Id:       i.ID,
			Title:    i.Title,
			Artists:  i.Artists,
			Album:    i.Album,
			TrackNum: i.TrackNum,
			DiscNum:  i.DiscNum,
		})
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (e *MusicPlayerEndpoint) GetNowPlayingHandler(w http.ResponseWriter, r *http.Request) {
	response := MusicData{
		Id:       e.mp.bot.currentAudioData.ID,
		Title:    e.mp.bot.currentAudioData.Title,
		Artists:  e.mp.bot.currentAudioData.Artists,
		Album:    e.mp.bot.currentAudioData.Album,
		TrackNum: e.mp.bot.currentAudioData.TrackNum,
		DiscNum:  e.mp.bot.currentAudioData.DiscNum,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (e *MusicPlayerEndpoint) StartPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	e.mp.StartPlaylist()
	w.WriteHeader(http.StatusOK)
}

func (e *MusicPlayerEndpoint) StopPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	e.mp.StopPlaylist()
	w.WriteHeader(http.StatusOK)
}
