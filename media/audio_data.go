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
	ret := ad.Title
	if ad.Artists != nil {
		artistsStrTrimmed := strings.TrimSpace(*ad.Artists)
		if artistsStrTrimmed != "" {
			ret = artistsStrTrimmed + " - " + ret
		}
	}
	if ad.Album != nil {
		albumStrTrimmed := strings.TrimSpace(*ad.Album)
		if albumStrTrimmed != "" {
			ret += " (from " + *ad.Album + ")"
		}
	}
	return ret
}
