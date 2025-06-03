package mediax

import (
	"fmt"
	"github.com/getevo/evo/v2/lib/db"
	"mediax/apps/media"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

var (
	Origins       map[string]*media.Origin
	VideoProfiles map[string]*media.VideoProfile
	Wait          sync.WaitGroup
)

func InitializeConfig() {
	Wait.Wait()
	Wait.Add(1)
	defer Wait.Done()
	var origins []media.Origin
	db.Preload("Project").Where("deleted_at IS NULL").Find(&origins)
	Origins = make(map[string]*media.Origin)
	var storages []media.Storage
	db.Order("priority ASC").Find(&storages)
	for idx, _ := range origins {
		var origin = origins[idx]
		for i, _ := range storages {
			if storages[i].ProjectID == origin.ProjectID {
				storages[i].Init()
				origin.Storages = append(origin.Storages, &storages[i])
			}
		}
		Origins[strings.ToLower(origin.Domain)] = &origin
	}

	var videoProfiles []media.VideoProfile
	db.Find(&videoProfiles)
	VideoProfiles = make(map[string]*media.VideoProfile)
	for idx, _ := range videoProfiles {
		vp := videoProfiles[idx]
		VideoProfiles[vp.Profile] = &vp
	}
}

func GetURLExtension(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	ext := filepath.Ext(path.Base(parsedURL.Path))
	if len(ext) > 0 {
		ext = strings.ToLower(ext[1:])
	}
	return ext, nil
}
