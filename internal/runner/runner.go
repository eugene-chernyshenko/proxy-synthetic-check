package runner

import (
	"log"
	"net/http"
	"time"

	"eugene-chernyshenko/proxy-synthetic-check/internal/config"
	"eugene-chernyshenko/proxy-synthetic-check/internal/metrics"
	"eugene-chernyshenko/proxy-synthetic-check/internal/proxy"
	"eugene-chernyshenko/proxy-synthetic-check/internal/request"
)

// Run starts a proxy runner that sends requests at specified interval
func Run(m *metrics.Metrics, proxyID string, proxyConfig config.Proxy, targetURL string, requestInterval, requestTimeout time.Duration) {
	// Create transport for this proxy
	transport, err := proxy.CreateTransport(proxyConfig.Protocol, proxyConfig.Proxy)
	if err != nil {
		log.Fatalf("[%s] Error creating proxy transport: %v", proxyID, err)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
	}

	log.Printf("[%s] Starting proxy runner (protocol: %s, proxy: %s)", proxyID, proxyConfig.Protocol, proxy.MaskAuth(proxyConfig.Protocol, proxyConfig.Proxy))

	// Create ticker for this proxy
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()

	// Send initial request immediately
	go request.Make(m, client, targetURL, proxyID, proxyConfig.Protocol, proxyConfig.Labels)

	// Send requests at intervals
	for range ticker.C {
		go request.Make(m, client, targetURL, proxyID, proxyConfig.Protocol, proxyConfig.Labels)
	}
}

