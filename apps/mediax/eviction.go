package mediax

import (
	"github.com/getevo/evo/v2/lib/log"
	"mediax/apps/media"
	"time"
)

// startEvictionLoop launches a background goroutine that periodically checks
// every project's cache directory and removes the oldest files when the
// configured cache size limit is exceeded.
// It also runs once immediately on startup so the cache is clean from the start.
func startEvictionLoop() {
	go func() {
		runEviction()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			runEviction()
		}
	}()
}

// runEviction iterates over all currently-loaded projects (under read-lock),
// reports the current cache size to Prometheus, and evicts files when over limit.
func runEviction() {
	mu.RLock()
	type projectInfo struct {
		name     string
		cacheDir string
		maxBytes int64
	}
	seen := map[int]bool{}
	var projects []projectInfo

	for _, o := range Origins {
		if o.Project == nil || seen[o.ProjectID] {
			continue
		}
		seen[o.ProjectID] = true
		if o.Project.CacheDir == "" || o.Project.CacheSize == "" {
			continue
		}
		maxBytes, err := media.ParseCacheSize(o.Project.CacheSize)
		if err != nil || maxBytes == 0 {
			continue
		}
		projects = append(projects, projectInfo{
			name:     o.Project.Name,
			cacheDir: o.Project.CacheDir,
			maxBytes: maxBytes,
		})
	}
	mu.RUnlock()

	for _, p := range projects {
		// Report current size before eviction.
		if sz, err := media.DirSize(p.cacheDir); err == nil {
			media.MetricCacheSizeBytes.WithLabelValues(p.name).Set(float64(sz))
		}

		removed, freed, err := media.EvictCache(p.cacheDir, p.maxBytes)
		if err != nil {
			log.Error("cache eviction failed", "project", p.name, "cache_dir", p.cacheDir, "error", err)
			continue
		}
		if removed > 0 {
			log.Info("cache eviction completed",
				"project", p.name,
				"files_removed", removed,
				"bytes_freed", freed,
			)
			media.MetricCacheEvictedFilesTotal.WithLabelValues(p.name).Add(float64(removed))
			media.MetricCacheEvictedBytesTotal.WithLabelValues(p.name).Add(float64(freed))

			// Update the gauge to reflect the post-eviction size.
			if sz, err := media.DirSize(p.cacheDir); err == nil {
				media.MetricCacheSizeBytes.WithLabelValues(p.name).Set(float64(sz))
			}
		}
	}
}
