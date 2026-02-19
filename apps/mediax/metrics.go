package mediax

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// metricRequests counts every request served, labelled by file extension and outcome.
	metricRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "mediax",
		Name:      "requests_total",
		Help:      "Total number of media requests handled.",
	}, []string{"extension", "status"})

	// metricProcessingDuration records how long the encoder Processor takes.
	// Only recorded when an encoder Processor is actually invoked (not for pass-through).
	metricProcessingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "mediax",
		Name:      "processing_duration_seconds",
		Help:      "Histogram of encoder processing durations in seconds.",
		Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"extension"})
)
