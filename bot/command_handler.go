// FIXME: Locking and unlocking mutex is not fun

package bot

import (
	"fmt"
	"html"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/EricZhang456/mumble-music-bot/media"
	"github.com/EricZhang456/mumble-music-bot/utils"
	"github.com/google/shlex"
	"gorm.io/gorm"
)

type CommandHandler interface {
	HandleCommand(commandRaw string) *string
}

type MusicPlayerCommandHandler struct {
	mp            *MusicPlayer
	db            *gorm.DB
	commandPrefix string
	allTrackPages [][]media.AudioData
}

func CreateCommandHandler(commandPrefix string, mp *MusicPlayer, db *gorm.DB) *MusicPlayerCommandHandler {
	commandHandler := &MusicPlayerCommandHandler{mp: mp, db: db, commandPrefix: commandPrefix}
	if err := commandHandler.populatePages(5); err != nil {
		log.Fatal("Cannot populate pages for command handler.")
	}
	return commandHandler
}

func (com *MusicPlayerCommandHandler) populatePages(pageSize int) error {
	var allAudioData []media.AudioData
	if err := com.db.Find(&allAudioData).Error; err != nil {
		return err
	}
	numPages := int(math.Ceil(float64(len(allAudioData)) / float64(pageSize)))
	com.allTrackPages = make([][]media.AudioData, 0, numPages)
	for i := 0; i < len(allAudioData); i += pageSize {
		end := min(i+pageSize, len(allAudioData))
		com.allTrackPages = append(com.allTrackPages, allAudioData[i:end])
	}
	return nil
}

func (com *MusicPlayerCommandHandler) HandleCommand(commandRaw string) *string {
	commandRawUnescape := html.UnescapeString(commandRaw)
	command := strings.TrimPrefix(strings.TrimSpace(commandRawUnescape), com.commandPrefix)
	commandParts, err := shlex.Split(command)
	if err != nil {
		return nil
	}
	verb := commandParts[0]
	args := commandParts[1:]
	// i hate this
	switch verb {
	case "help":
		result := com.replyHelp()
		return &result
	case "tracks":
		result := com.getTracks(args)
		return &result
	case "add":
		result := com.addTrack(args)
		return &result
	case "addalbum":
		result := com.addAlbum(args)
		return &result
	case "mode":
		result := com.setOrGetMode(args)
		return &result
	case "remove":
		result := com.removeFromPlaylist(args)
		return &result
	case "skip":
		result := com.skipTrack()
		return &result
	case "playlist":
		result := com.replyPlaylist()
		return &result
	case "nowplaying":
		result := com.replyNowPlaying()
		return &result
	case "start":
		result := com.startPlaylist()
		return &result
	case "stop":
		result := com.stopPlaylist()
		return &result
	case "clear":
		result := com.clearPlaylist()
		return &result
	}
	return nil
}

// i don't like this either
func (com *MusicPlayerCommandHandler) replyHelp() string {
	ret := "<b>Available Commands:</b><br>"
	ret += fmt.Sprintf("<b>%shelp:</b> Show this help message.<br>", com.commandPrefix)
	ret += fmt.Sprintf("<b>%stracks <i>&lt;page number&gt;</i>:</b> Show available tracks. "+
		"Invoke with no arguments to show the first page.<br>", com.commandPrefix)
	ret += fmt.Sprintf("<b>%sadd <i>&lt;track id&gt;</i>:</b> Add a track to playlist by its track ID.<br>", com.commandPrefix)
	ret += fmt.Sprintf("<b>%saddalbum <i>&lt;album name&gt;</i>:</b> Add an entire album to playlist.<br>", com.commandPrefix)
	ret += fmt.Sprintf("<b>%smode <i>&lt;playback mode&gt;</i>:</b> Set playback mode. "+
		"Invoke with no arguments to see current plaback mode. "+
		"Available values are: &quot;single&quot;, &quot;shuffle&quot;, &quot;repeat&quot;, &quot;shufflerepeat&quot;.<br>", com.commandPrefix)
	ret += fmt.Sprintf("<b>%sremove <i>&lt;index&gt;</i>:</b> Remove a track from playlist by its index in the playlist.<br>", com.commandPrefix)
	ret += fmt.Sprintf("<b>%sskip:</b> Skip the current track.<br>", com.commandPrefix)
	ret += fmt.Sprintf("<b>%snowplaying:</b> Show what's playing right now.<br>", com.commandPrefix)
	ret += fmt.Sprintf("<b>%splaylist:</b> Show the current playlist.<br>", com.commandPrefix)
	ret += fmt.Sprintf("<b>%sstart:</b> Start playback.<br>", com.commandPrefix)
	ret += fmt.Sprintf("<b>%sstop:</b> Stop playback and rewind to first track in playlist.<br>", com.commandPrefix)
	ret += fmt.Sprintf("<b>%sclear:</b> Stop playback and clear playlist.", com.commandPrefix)
	return ret
}

func (com *MusicPlayerCommandHandler) getTracks(args []string) string {
	var pageNum int
	if len(args) == 0 {
		pageNum = 1
	} else {
		var err error
		pageNum, err = strconv.Atoi(args[0])
		if err != nil {
			return "Not a valid page number."
		}
	}
	ret := fmt.Sprintf("<b>Showing page %d of %d</b>:<br>", pageNum, len(com.allTrackPages))
	page := com.allTrackPages[pageNum-1]
	for _, i := range page {
		ret += fmt.Sprintf("<b>%d:</b> %s<br>", i.ID, i.ToString())
	}
	ret += fmt.Sprintf("<br>Type <b>%stracks %d</b> to see the next page.", com.commandPrefix, pageNum+1)
	return ret
}

func (com *MusicPlayerCommandHandler) replyNowPlaying() string {
	// ehh whatever
	com.mp.mu.Lock()
	current := com.mp.bot.currentAudioData
	com.mp.mu.Unlock()
	if current == nil {
		return "Not playing anything right now."
	}
	return "<b>Now playing:</b> " + current.ToString()
}

func (com *MusicPlayerCommandHandler) replyPlaylist() string {
	com.mp.mu.Lock()
	defer com.mp.mu.Unlock()
	ret := "<b>Current playlist:</b><br>"
	for index, i := range com.mp.playlist {
		ret += fmt.Sprintf("<b>%d:</b> %s", index+1, i.ToString())
		if index != len(com.mp.playlist)-1 {
			ret += "<br>"
		}
	}
	return ret
}

func (com *MusicPlayerCommandHandler) getTrackById(id uint) (*media.AudioData, error) {
	var track media.AudioData
	if err := com.db.First(&track, id).Error; err != nil {
		return nil, err
	}
	return &track, nil
}

func (com *MusicPlayerCommandHandler) addTrack(args []string) string {
	trackId, err := strconv.Atoi(args[0])
	if err != nil {
		return "Invalid track ID."
	}
	track, err := com.getTrackById(uint(trackId))
	if err != nil {
		return "Invalid track ID."
	}
	com.mp.AddToPlaylist(*track)
	return "<b>Adding track:</b> " + track.ToString()
}

func (com *MusicPlayerCommandHandler) addAlbum(args []string) string {
	if len(args) == 0 {
		return "Album name needed."
	}
	var albums []string
	if err := com.db.Model(&media.AudioData{}).Distinct("album").Where("album IS NOT NULL").Pluck("album", &albums).Error; err != nil {
		return "Error when searching for albums."
	}
	if len(albums) == 0 {
		return "No albums are in DB somehow."
	}

	bestAlbum := ""
	bestDistance := math.MaxInt
	for _, a := range albums {
		dist := utils.Levenshtein(strings.ToLower(args[0]), strings.ToLower(a))
		if dist < bestDistance {
			bestDistance = dist
			bestAlbum = a
		}
	}
	if bestAlbum == "" {
		return "No matching album found."
	}

	var tracks []media.AudioData
	if err := com.db.
		Where("album = ?", bestAlbum).
		Order("CASE WHEN disc_num IS NULL THEN 1 ELSE 0 END, disc_num").
		Order("CASE WHEN track_num IS NULL THEN 1 ELSE 0 END, track_num").
		Order("title COLLATE NOCASE ASC").
		Find(&tracks).Error; err != nil {
		return "Database error while fetching album tracks."
	}
	if len(tracks) == 0 {
		return "No tracks found for album " + bestAlbum
	}

	com.mp.AddAllToPlaylist(tracks)
	return fmt.Sprintf("Adding album <b>%s</b> to playlist. (%d tracks)", bestAlbum, len(tracks))
}

func (com *MusicPlayerCommandHandler) setOrGetMode(args []string) string {
	if len(args) == 0 {
		return "<b>Current playback mode:</b> " + PlaybackModeToString(com.mp.GetMode())
	}
	modeStr := strings.ToLower(args[0])
	var mode PlaybackMode
	switch modeStr {
	case "single":
		mode = Single
	case "shuffle":
		mode = Shuffle
	case "repeat":
		mode = Repeat
	case "shufflerepeat":
		mode = ShuffleRepeat
	default:
		return "Invalid playback mode: " + modeStr
	}
	com.mp.SetMode(mode)
	return "<b>Changed playback mode to:</b> " + PlaybackModeToString(mode)
}

func (com *MusicPlayerCommandHandler) removeFromPlaylist(args []string) string {
	com.mp.mu.Lock()
	if len(com.mp.playlist) == 0 {
		com.mp.mu.Unlock()
		return "Playlist is empty."
	}
	com.mp.mu.Unlock()
	playlistIndex, err := strconv.Atoi(args[0])
	if err != nil {
		return "Invalid index."
	}
	playlistIndex--

	removedTrack, result := com.mp.RemoveFromPlaylist(playlistIndex)
	switch result {
	case Success:
		return fmt.Sprintf("Removed track <b>%d: %s</b> from playlist.", playlistIndex+1, removedTrack.Title)
	case Playing:
		return "You can't remove the track that's currently playing."
	case OutOfRange:
		return "Invalid index."
	}
	return ""
}

func (com *MusicPlayerCommandHandler) skipTrack() string {
	com.mp.mu.Lock()
	if len(com.mp.playlist) == 0 {
		com.mp.mu.Unlock()
		return "Playlist is empty."
	}
	com.mp.mu.Unlock()
	if com.mp.Skip() {
		return "Skipping track."
	}
	return "Not playing anything right now."
}

func (com *MusicPlayerCommandHandler) startPlaylist() string {
	com.mp.mu.Lock()
	if len(com.mp.playlist) == 0 {
		com.mp.mu.Unlock()
		return "Playlist is empty."
	}
	com.mp.mu.Unlock()
	com.mp.StartPlaylist()
	return "Starting playback."
}

func (com *MusicPlayerCommandHandler) stopPlaylist() string {
	com.mp.mu.Lock()
	if len(com.mp.playlist) == 0 {
		com.mp.mu.Unlock()
		return "Playlist is empty."
	}
	com.mp.mu.Unlock()
	com.mp.StopPlaylist()
	return "Stopping playback."
}

func (com *MusicPlayerCommandHandler) clearPlaylist() string {
	com.mp.mu.Lock()
	if len(com.mp.playlist) == 0 {
		com.mp.mu.Unlock()
		return "Playlist is empty."
	}
	com.mp.mu.Unlock()
	com.mp.ClearPlaylist()
	return "Stopping playback and clearing playlist."
}
