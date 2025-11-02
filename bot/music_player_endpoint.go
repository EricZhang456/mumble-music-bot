package bot

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/EricZhang456/mumble-music-bot/media"
	"gorm.io/gorm"
)

type MusicQueryResponse struct {
	Id       uint    `json:"id"`
	Title    string  `json:"title"`
	Artists  *string `json:"artists,omitempty"`
	Album    *string `json:"album,omitempty"`
	TrackNum *int    `json:"track_num,omitempty"`
}

type AddAllRequest struct {
	Ids []uint `json:"ids"`
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
	response := make([]MusicQueryResponse, 0, len(audioData))
	for _, t := range audioData {
		response = append(response, MusicQueryResponse{
			Id:       t.ID,
			Title:    t.Title,
			Artists:  t.Artists,
			Album:    t.Album,
			TrackNum: t.TrackNum,
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

func (e *MusicPlayerEndpoint) StartPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	e.mp.StartPlaylist()
	w.WriteHeader(http.StatusOK)
}

func (e *MusicPlayerEndpoint) StopPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	e.mp.StopPlaylist()
	w.WriteHeader(http.StatusOK)
}
