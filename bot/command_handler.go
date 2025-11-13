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
	pageSize      int
	commandPrefix string
	allTrackPages [][]media.AudioData
	allTracks     []media.AudioData
	allAlbums     []string
}

func CreateCommandHandler(commandPrefix string, mp *MusicPlayer, db *gorm.DB) *MusicPlayerCommandHandler {
	commandHandler := &MusicPlayerCommandHandler{mp: mp, db: db, commandPrefix: commandPrefix}
	if err := commandHandler.populatePages(5); err != nil {
		log.Fatal("Cannot populate pages for command handler.")
	}
	if err := commandHandler.populateAlbums(); err != nil {
		log.Fatal("Cannot fetch albums from DB.")
	}
	return commandHandler
}

func (com *MusicPlayerCommandHandler) populatePages(pageSize int) error {
	if err := com.db.Find(&com.allTracks).Error; err != nil {
		return err
	}

	numPages := int(math.Ceil(float64(len(com.allTracks)) / float64(pageSize)))
	com.pageSize = pageSize
	com.allTrackPages = make([][]media.AudioData, 0, numPages)

	for i := 0; i < len(com.allTracks); i += pageSize {
		end := min(i+pageSize, len(com.allTracks))
		com.allTrackPages = append(com.allTrackPages, com.allTracks[i:end])
	}
	return nil
}

func (com *MusicPlayerCommandHandler) populateAlbums() error {
	var albums []string
	if err := com.db.Model(&media.AudioData{}).Distinct("album").Where("album IS NOT NULL").Pluck("album", &albums).Error; err != nil {
		return err
	}
	com.allAlbums = albums
	return nil
}

func (com *MusicPlayerCommandHandler) HandleCommand(commandRaw string) *string {
	commandRawUnescape := html.UnescapeString(commandRaw)
	command := strings.TrimPrefix(strings.TrimSpace(commandRawUnescape), com.commandPrefix)
	commandParts, err := shlex.Split(command)
	if err != nil {
		return nil
	}
	verb := strings.ToLower(commandParts[0])
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
	case "pause":
		result := com.pauseToggle()
		return &result
	}
	return nil
}

// i don't like this either
func (com *MusicPlayerCommandHandler) replyHelp() string {
	var sb strings.Builder
	sb.WriteString("<br><b>Available Commands:</b><br>")
	sb.WriteString(fmt.Sprintf("<b>%shelp:</b> Show this help message.<br>", com.commandPrefix))
	sb.WriteString(fmt.Sprintf("<b>%stracks <i>&lt;page number&gt;</i>:</b> Show available tracks. "+
		"Invoke with no arguments to show the first page.<br>", com.commandPrefix))
	sb.WriteString(fmt.Sprintf("<b>%sadd <i>&lt;track id&gt;</i>:</b> Add a track to playlist by its track ID.<br>", com.commandPrefix))
	sb.WriteString(fmt.Sprintf("<b>%saddalbum <i>&lt;album name&gt;</i>:</b> Add an entire album to playlist.<br>", com.commandPrefix))
	sb.WriteString(fmt.Sprintf("<b>%smode <i>&lt;playback mode&gt;</i>:</b> Set playback mode. "+
		"Invoke with no arguments to see current plaback mode. "+
		"Available values are: &quot;single&quot;, &quot;shuffle&quot;, &quot;repeat&quot;, &quot;shufflerepeat&quot;.<br>", com.commandPrefix))
	sb.WriteString(fmt.Sprintf("<b>%sremove <i>&lt;index&gt;</i>:</b> Remove a track from playlist by its index in the playlist.<br>", com.commandPrefix))
	sb.WriteString(fmt.Sprintf("<b>%sskip:</b> Skip the current track.<br>", com.commandPrefix))
	sb.WriteString(fmt.Sprintf("<b>%snowplaying:</b> Show what's playing right now.<br>", com.commandPrefix))
	sb.WriteString(fmt.Sprintf("<b>%splaylist:</b> Show the current playlist.<br>", com.commandPrefix))
	sb.WriteString(fmt.Sprintf("<b>%sstart:</b> Start playback.<br>", com.commandPrefix))
	sb.WriteString(fmt.Sprintf("<b>%sstop:</b> Stop playback and rewind to the first track in playlist.<br>", com.commandPrefix))
	sb.WriteString(fmt.Sprintf("<b>%spause:</b> Pause/Unpause playback.<br>", com.commandPrefix))
	sb.WriteString(fmt.Sprintf("<b>%sclear:</b> Stop playback and clear playlist.", com.commandPrefix))
	return sb.String()
}

func (com *MusicPlayerCommandHandler) getTracks(args []string) string {
	if len(com.allTracks) == 0 {
		return "No tracks available."
	}
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
	if pageNum <= 0 || pageNum > len(com.allTrackPages) {
		return "Page number out of range."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<br><b>Showing page %d of %d</b>:<br>", pageNum, len(com.allTrackPages)))
	page := com.allTrackPages[pageNum-1]
	startIndex := (pageNum - 1) * com.pageSize
	for index, i := range page {
		sb.WriteString(fmt.Sprintf("<b>%d:</b> %s<br>", startIndex+index+1, i.ToString()))
	}
	if pageNum != len(com.allTrackPages) {
		sb.WriteString(fmt.Sprintf("<br>Type <b>%stracks %d</b> to see the next page.", com.commandPrefix, pageNum+1))
	}
	return sb.String()
}

func (com *MusicPlayerCommandHandler) replyNowPlaying() string {
	current := com.mp.GetCurrentTrack()
	if current == nil {
		return "Not playing anything right now."
	}
	var sb strings.Builder
	sb.WriteString("<b>Now playing:</b> ")
	sb.WriteString(current.ToString())
	if com.mp.IsPaused() {
		sb.WriteString(" <i>(Paused)</i>")
	}
	return sb.String()
}

func (com *MusicPlayerCommandHandler) replyPlaylist() string {
	nowPlaying := com.mp.GetCurrentTrack()
	playlist := com.mp.GetPlaylist()
	if len(playlist) == 0 {
		return "Playlist is empty."
	}
	var sb strings.Builder
	sb.WriteString("<br><b>Current playlist:</b><br>")
	for index, i := range playlist {
		sb.WriteString(fmt.Sprintf("<b>%d:</b> %s", index+1, i.ToString()))
		if i == nowPlaying {
			sb.WriteString(" <i>(Now playing)</i>")
			if com.mp.IsPaused() {
				sb.WriteString(" <i>(Paused)</i>")
			}
		}
		if index != len(com.mp.playlist)-1 {
			sb.WriteString("<br>")
		}
	}
	return sb.String()
}

func (com *MusicPlayerCommandHandler) addTrack(args []string) string {
	trackId, err := strconv.Atoi(args[0])
	if err != nil || trackId <= 0 || trackId > len(com.allTracks) {
		return "Invalid track ID."
	}
	track := com.allTracks[trackId-1]
	com.mp.AddToPlaylist(track)
	return "<b>Adding track:</b> " + track.ToString()
}

func (com *MusicPlayerCommandHandler) addAlbum(args []string) string {
	if len(args) == 0 {
		return "Album name needed."
	}
	if len(com.allAlbums) == 0 {
		return "No albums are in DB somehow."
	}

	bestAlbum := ""
	bestDistance := math.MaxInt
	for _, a := range com.allAlbums {
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
	if len(com.mp.GetPlaylist()) == 0 {
		return "Playlist is empty."
	}
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
	if len(com.mp.GetPlaylist()) == 0 {
		return "Playlist is empty."
	}
	if err := com.mp.Skip(); err == nil {
		return "Skipping track."
	}
	return "Not playing anything right now."
}

func (com *MusicPlayerCommandHandler) startPlaylist() string {
	if len(com.mp.GetPlaylist()) == 0 {
		return "Playlist is empty."
	}
	com.mp.StartPlaylist()
	return "Starting playback."
}

func (com *MusicPlayerCommandHandler) stopPlaylist() string {
	if len(com.mp.GetPlaylist()) == 0 {
		return "Playlist is empty."
	}
	com.mp.StopPlaylist()
	return "Stopping playback."
}

func (com *MusicPlayerCommandHandler) clearPlaylist() string {
	if len(com.mp.GetPlaylist()) == 0 {
		return "Playlist is empty."
	}
	com.mp.ClearPlaylist()
	return "Stopping playback and clearing playlist."
}

func (com *MusicPlayerCommandHandler) pauseToggle() string {
	if com.mp.GetCurrentTrack() == nil {
		return "Playback is stopped."
	}
	paused := com.mp.IsPaused()
	if !paused {
		com.mp.Pause()
		return "Pausing audio."
	}
	com.mp.Unpause()
	return "Unpausing audio."
}
