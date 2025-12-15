package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"eugene-chernyshenko/proxy-synthetic-check/internal/config"
	"eugene-chernyshenko/proxy-synthetic-check/internal/metrics"
	"eugene-chernyshenko/proxy-synthetic-check/internal/runner"
)

func main() {
	// Load YAML config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading proxy configuration: %v", err)
	}

	// Initialize metrics with collected label keys
	buckets := cfg.GetLatencyBuckets()
	m := metrics.New(cfg.Proxies, buckets)
	log.Printf("Using latency buckets: %v", buckets)

	targetURL := cfg.TargetURL
	requestInterval := time.Duration(cfg.RequestInterval) * time.Millisecond
	requestTimeout := time.Duration(cfg.RequestTimeout) * time.Second

	// Default metrics port to 8080 if not specified
	metricsPort := cfg.MetricsPort
	if metricsPort == 0 {
		metricsPort = 8080
	}

	// Start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		addr := ":" + strconv.Itoa(metricsPort)
		log.Printf("Metrics server starting on %s/metrics", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("Error starting metrics server: %v", err)
		}
	}()

	log.Printf("Configuration:")
	log.Printf("  Target URL: %s", targetURL)
	log.Printf("  Request interval: %v", requestInterval)
	log.Printf("  Request timeout: %v", requestTimeout)
	log.Printf("  Metrics port: %d", metricsPort)
	log.Printf("  Number of proxies: %d", len(cfg.Proxies))

	// Start each proxy in a separate goroutine with sequential ID
	for i, proxyConfig := range cfg.Proxies {
		proxyID := "proxy_" + strconv.Itoa(i+1)
		go runner.Run(m, proxyID, proxyConfig, targetURL, requestInterval, requestTimeout)
	}

	// Keep main goroutine alive
	select {}
}
