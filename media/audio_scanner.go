package media

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/EricZhang456/mumble-music-bot/utils"
	"github.com/dhowden/tag"
	"gorm.io/gorm"
)

type AudioScanner struct {
	db *gorm.DB
}

var audioExtensions = map[string]struct{}{
	".aac": {}, ".flac": {}, ".m4a": {}, ".mp3": {}, ".ogg": {}, ".oga": {},
}

func CreateAudioScanner(db *gorm.DB) *AudioScanner {
	scanner := &AudioScanner{db: db}
	return scanner
}

func getAllAudioFiles(path string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if _, ok := audioExtensions[ext]; ok {
				files = append(files, path)
			}
		}
		return nil
	})
	return files, err
}

func hasMetadataChanged(existing, scanned AudioData) bool {
	return existing.Title != scanned.Title ||
		!utils.EqualPtr(existing.Artists, scanned.Artists) ||
		!utils.EqualPtr(existing.Album, scanned.Album) ||
		!utils.EqualPtr(existing.TrackNum, scanned.TrackNum) ||
		!utils.EqualPtr(existing.DiscNum, scanned.DiscNum)
}

func getMetadata(path string) (*AudioData, error) {
	fullpath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tags, err := tag.ReadFrom(f)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(tags.Title())
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	var artists, album *string
	if a := tags.Artist(); a != "" {
		artists = &a
	}
	if al := tags.Album(); al != "" {
		album = &al
	}
	var trackNum, discNum *int
	if tn, _ := tags.Track(); tn != 0 {
		trackNum = &tn
	}
	if dn, _ := tags.Disc(); dn != 0 {
		discNum = &dn
	}
	return &AudioData{
		Path:     fullpath,
		Title:    title,
		Artists:  artists,
		Album:    album,
		TrackNum: trackNum,
		DiscNum:  discNum,
	}, nil
}

func (ms *AudioScanner) ScanAndWriteToDb(path string) error {
	filesInDb := []AudioData{}
	if err := ms.db.Find(&filesInDb).Error; err != nil {
		return err
	}

	pathToAudio := make(map[string]AudioData)
	for _, f := range filesInDb {
		pathToAudio[f.Path] = f
	}

	seen := make(map[string]struct{})
	var toAdd, toUpdate, toDelete []AudioData

	files, err := getAllAudioFiles(path)
	if err != nil {
		return err
	}

	for _, f := range files {
		seen[f] = struct{}{}
		meta, err := getMetadata(f)
		if err != nil {
			continue
		}
		if existing, ok := pathToAudio[f]; !ok {
			toAdd = append(toAdd, *meta)
		} else if hasMetadataChanged(existing, *meta) {
			meta.ID = existing.ID
			toUpdate = append(toUpdate, *meta)
		}
	}

	for _, f := range filesInDb {
		if _, ok := seen[f.Path]; !ok {
			toDelete = append(toDelete, f)
		}
	}

	return ms.db.Transaction(func(tx *gorm.DB) error {
		if len(toAdd) > 0 {
			if err := tx.Create(&toAdd).Error; err != nil {
				return err
			}
		}
		if len(toUpdate) > 0 {
			if err := tx.Save(&toUpdate).Error; err != nil {
				return err
			}
		}
		if len(toDelete) > 0 {
			for _, del := range toDelete {
				if err := tx.Delete(&del).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
