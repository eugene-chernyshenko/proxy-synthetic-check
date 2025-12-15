package metrics

import (
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	"eugene-chernyshenko/proxy-synthetic-check/internal/config"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	LabelKeys       []string
}

// New creates and initializes Prometheus metrics with collected label keys
func New(proxies []config.Proxy, buckets []float64) *Metrics {
	// Collect all unique label keys from all proxies
	labelKeys := collectLabelKeys(proxies)

	// Build label list: proxy_id, proxy_protocol, ...labelKeys..., status, error
	requestsLabels := []string{"proxy_id", "proxy_protocol"}
	requestsLabels = append(requestsLabels, labelKeys...)
	requestsLabels = append(requestsLabels, "status", "error")

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "requests_total",
			Help: "Total number of requests",
		},
		requestsLabels,
	)

	// Build label list for histogram: proxy_id, proxy_protocol, ...labelKeys...
	durationLabels := []string{"proxy_id", "proxy_protocol"}
	durationLabels = append(durationLabels, labelKeys...)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "Request latency distribution",
			Buckets: buckets,
		},
		durationLabels,
	)

	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(requestDuration)

	return &Metrics{
		RequestsTotal:   requestsTotal,
		RequestDuration: requestDuration,
		LabelKeys:       labelKeys,
	}
}

// collectLabelKeys collects all unique label keys from all proxies
func collectLabelKeys(proxies []config.Proxy) []string {
	keySet := make(map[string]bool)
	for _, p := range proxies {
		for key := range p.Labels {
			keySet[key] = true
		}
	}

	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}

	// Sort keys for consistent order
	sort.Strings(keys)

	return keys
}

