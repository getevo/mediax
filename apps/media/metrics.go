package media

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MetricCacheSizeBytes reports the current total cache directory size per project.
	// Updated by the eviction loop.
	MetricCacheSizeBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "mediax",
		Name:      "cache_size_bytes",
		Help:      "Current cache directory size in bytes.",
	}, []string{"project"})

	// MetricCacheEvictedFilesTotal counts the number of files removed by cache eviction.
	MetricCacheEvictedFilesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "mediax",
		Name:      "cache_evicted_files_total",
		Help:      "Total number of files removed by cache eviction.",
	}, []string{"project"})

	// MetricCacheEvictedBytesTotal counts the number of bytes freed by cache eviction.
	MetricCacheEvictedBytesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "mediax",
		Name:      "cache_evicted_bytes_total",
		Help:      "Total bytes freed by cache eviction.",
	}, []string{"project"})
)
