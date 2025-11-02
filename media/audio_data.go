package media

import (
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
