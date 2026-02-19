package media

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/getevo/evo/v2/lib/log"
)

// ParseCacheSize converts a human-readable size string (e.g. "1 GB", "500MB", "10gb")
// to bytes. Supported suffixes: B, KB, MB, GB, TB (case-insensitive).
// Returns 0, nil for empty or "0" input (meaning no limit).
func ParseCacheSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" || s == "0" {
		return 0, nil
	}

	// Check longest suffix first to avoid "GB" matching "B".
	multipliers := []struct {
		suffix string
		mult   int64
	}{
		{"TB", 1 << 40},
		{"GB", 1 << 30},
		{"MB", 1 << 20},
		{"KB", 1 << 10},
		{"B", 1},
	}

	for _, m := range multipliers {
		if strings.HasSuffix(s, m.suffix) {
			numStr := strings.TrimSpace(strings.TrimSuffix(s, m.suffix))
			var n float64
			if _, err := fmt.Sscanf(numStr, "%f", &n); err != nil || n < 0 {
				return 0, fmt.Errorf("invalid cache size %q", s)
			}
			return int64(n * float64(m.mult)), nil
		}
	}

	// No recognised suffix — treat as bare bytes.
	var n int64
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil || n < 0 {
		return 0, fmt.Errorf("invalid cache size %q: no unit suffix", s)
	}
	return n, nil
}

// DirSize returns the total size of all regular files under dir.
func DirSize(dir string) (int64, error) {
	var total int64
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err == nil {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

// EvictCache removes the oldest files in dir until the total size is ≤ maxBytes.
// Lock files (*.lock) and directories are never removed.
// Returns the number of files removed and total bytes freed.
func EvictCache(dir string, maxBytes int64) (removed int, freed int64, err error) {
	if maxBytes <= 0 {
		return 0, 0, nil
	}

	type entry struct {
		path string
		size int64
		mod  time.Time
	}

	var entries []entry
	var total int64

	walkErr := filepath.WalkDir(dir, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil || d.IsDir() {
			return nil
		}
		// Never evict active lock files — they mark in-progress downloads.
		if strings.HasSuffix(p, ".lock") {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}
		entries = append(entries, entry{path: p, size: info.Size(), mod: info.ModTime()})
		total += info.Size()
		return nil
	})
	if walkErr != nil {
		return 0, 0, fmt.Errorf("cache eviction walk error: %w", walkErr)
	}

	if total <= maxBytes {
		return 0, 0, nil // already within limit
	}

	// Evict oldest files first.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].mod.Before(entries[j].mod)
	})

	for _, e := range entries {
		if total <= maxBytes {
			break
		}
		if removeErr := os.Remove(e.path); removeErr != nil {
			log.Warning("cache eviction: failed to remove file", "path", e.path, "error", removeErr)
			continue
		}
		total -= e.size
		freed += e.size
		removed++
	}

	return removed, freed, nil
}
