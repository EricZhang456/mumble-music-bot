package media

import (
	"strings"

	"gorm.io/gorm"
)

type AudioData struct {
	gorm.Model
	Path     string
	Title    string
	Artists  *string
	Album    *string
	TrackNum *int
	DiscNum  *int
}

func (ad AudioData) ToString() string {
	var sb strings.Builder
	sb.WriteString(ad.Title)
	if ad.Artists != nil {
		artistsStrTrimmed := strings.TrimSpace(*ad.Artists)
		if artistsStrTrimmed != "" {
			sb.Reset()
			sb.WriteString(artistsStrTrimmed)
			sb.WriteString(" - ")
			sb.WriteString(ad.Title)
		}
	}
	if ad.Album != nil {
		albumStrTrimmed := strings.TrimSpace(*ad.Album)
		if albumStrTrimmed != "" {
			sb.WriteString("(from ")
			sb.WriteString(albumStrTrimmed)
			sb.WriteString(")")
		}
	}
	return sb.String()
}
