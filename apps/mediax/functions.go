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
	// mu protects Origins and VideoProfiles for concurrent read access.
	// InitializeConfig holds a write lock for its entire duration, so readers
	// always see a fully-consistent snapshot and never a partially-built map.
	mu sync.RWMutex

	// ready is closed exactly once after the first successful InitializeConfig,
	// unblocking all requests that arrived before the initial load completed.
	ready     = make(chan struct{})
	readyOnce sync.Once

	Origins       map[string]*media.Origin
	VideoProfiles map[string]*media.VideoProfile
)

func InitializeConfig() {
	// Write-lock for the full duration: this serializes concurrent reload calls
	// AND prevents readers from seeing a half-built map during the swap.
	mu.Lock()
	defer mu.Unlock()

	// Always signal readiness after the first call completes, even if a reload
	// is what triggered this call.
	defer readyOnce.Do(func() { close(ready) })

	var origins []media.Origin
	db.Preload("Project").Where("deleted_at IS NULL").Find(&origins)

	newOrigins := make(map[string]*media.Origin, len(origins))
	var storages []media.Storage
	db.Order("priority ASC").Find(&storages)
	for idx := range origins {
		origin := origins[idx]
		for i := range storages {
			if storages[i].ProjectID == origin.ProjectID {
				storages[i].Init()
				origin.Storages = append(origin.Storages, &storages[i])
			}
		}
		newOrigins[strings.ToLower(origin.Domain)] = &origin
	}

	var videoProfiles []media.VideoProfile
	db.Find(&videoProfiles)
	newVideoProfiles := make(map[string]*media.VideoProfile, len(videoProfiles))
	for idx := range videoProfiles {
		vp := videoProfiles[idx]
		newVideoProfiles[vp.Profile] = &vp
	}

	// Atomic swap: readers blocked by mu.RLock will see the new maps immediately
	// after this function returns.
	Origins = newOrigins
	VideoProfiles = newVideoProfiles
}

// lookupOrigin returns the Origin for a hostname under a read lock.
func lookupOrigin(host string) (*media.Origin, bool) {
	mu.RLock()
	defer mu.RUnlock()
	v, ok := Origins[host]
	return v, ok
}

// lookupVideoProfile returns a VideoProfile by name under a read lock.
func lookupVideoProfile(profile string) (*media.VideoProfile, bool) {
	mu.RLock()
	defer mu.RUnlock()
	v, ok := VideoProfiles[profile]
	return v, ok
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
