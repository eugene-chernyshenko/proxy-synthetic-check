package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/proxy"
	"gopkg.in/yaml.v3"
)

var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "requests_total",
			Help: "Total number of requests",
		},
		[]string{"proxy_type", "proxy_protocol", "status", "error"},
	)

	requestDuration *prometheus.HistogramVec
)

// ProxyConfig represents the YAML configuration file structure
type ProxyConfig struct {
	TargetURL       string    `yaml:"target_url"`
	RequestInterval int       `yaml:"request_interval_ms"`
	RequestTimeout  int       `yaml:"request_timeout"`
	MetricsPort     int       `yaml:"metrics_port"`
	LatencyBuckets  []float64 `yaml:"latency_buckets,omitempty"` // Optional custom buckets
	Proxies         []Proxy   `yaml:"proxies"`
}

// Proxy represents a single proxy configuration
type Proxy struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"` // socks5, http
	URL  string `yaml:"url"`
}

func init() {
	prometheus.MustRegister(requestsTotal)
}

// maskAuth hides password in URL for safe output
func maskAuth(s string) string {
	u, err := url.Parse(s)
	if err != nil {
		return s
	}
	if u.User != nil {
		u.User = url.User(u.User.Username())
	}
	return u.String()
}

func loadProxyConfig() (*ProxyConfig, error) {
	data, err := os.ReadFile("proxies.yaml")
	if err != nil {
		return nil, err
	}

	var config ProxyConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if len(config.Proxies) == 0 {
		return nil, errors.New("no proxies configured in config file")
	}

	return &config, nil
}

// getLatencyBuckets returns latency buckets, using config if provided, otherwise defaults
func getLatencyBuckets(config *ProxyConfig) []float64 {
	if len(config.LatencyBuckets) > 0 {
		return config.LatencyBuckets
	}
	// Default buckets with more observability in 0.2-2s range
	return []float64{
		0.05, // 50ms - very fast
		0.1,  // 100ms - fast
		0.2,  // 200ms - normal
		0.3,  // 300ms - added for better observability
		0.4,  // 400ms - added for better observability
		0.5,  // 500ms - acceptable
		0.75, // 750ms - added for better observability
		1.0,  // 1s - slow
		1.5,  // 1.5s - added for better observability
		2.0,  // 2s - very slow
		5.0,  // 5s - critically slow
		10.0, // 10s - near timeout
		30.0, // 30s - timeout
	}
}

// initRequestDurationMetric initializes the requestDuration metric with custom buckets
func initRequestDurationMetric(buckets []float64) {
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "Request latency distribution",
			Buckets: buckets,
		},
		[]string{"proxy_type", "proxy_protocol"},
	)
	prometheus.MustRegister(requestDuration)
}

// createProxyTransport creates HTTP transport based on proxy type
func createProxyTransport(proxyType, proxyURL string) (*http.Transport, error) {
	proxyURI, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	// Ensure scheme is present
	if proxyURI.Scheme == "" {
		if proxyType == "socks5" {
			proxyURL = "socks5://" + proxyURL
		} else if proxyType == "http" {
			proxyURL = "http://" + proxyURL
		}
		proxyURI, err = url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
	}

	switch strings.ToLower(proxyType) {
	case "socks5":
		proxyAddr := proxyURI.Host
		if proxyAddr == "" {
			return nil, errors.New("proxy address (host:port) is not specified")
		}

		var auth *proxy.Auth
		if proxyURI.User != nil {
			password, _ := proxyURI.User.Password()
			auth = &proxy.Auth{
				User:     proxyURI.User.Username(),
				Password: password,
			}
		}

		dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
		if err != nil {
			return nil, err
		}

		return &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
		}, nil

	case "http":
		// HTTP proxy using http.ProxyURL
		return &http.Transport{
			Proxy: http.ProxyURL(proxyURI),
		}, nil

	default:
		return nil, errors.New("unsupported proxy type: " + proxyType)
	}
}

// categorizeError categorizes errors into types for metrics
func categorizeError(err error) (errorType, httpStatusCode string) {
	if err == nil {
		return "", ""
	}

	errStr := err.Error()
	errLower := strings.ToLower(errStr)

	// Check for timeout errors
	if strings.Contains(errLower, "timeout") ||
		strings.Contains(errLower, "deadline exceeded") ||
		strings.Contains(errLower, "i/o timeout") {
		return "timeout", ""
	}

	// Check for DNS errors
	if strings.Contains(errLower, "no such host") ||
		strings.Contains(errLower, "dns") ||
		strings.Contains(errLower, "name resolution") {
		return "dns_error", ""
	}

	// Check for EOF errors (connection closed unexpectedly)
	if err == io.EOF || strings.Contains(errLower, "eof") {
		return "connection_error", ""
	}

	// Check for connection errors
	if strings.Contains(errLower, "connection refused") ||
		strings.Contains(errLower, "connection reset") ||
		strings.Contains(errLower, "broken pipe") ||
		strings.Contains(errLower, "network is unreachable") {
		return "connection_error", ""
	}

	// Check for URL errors (which might contain HTTP status codes)
	if urlErr, ok := err.(*url.Error); ok {
		if urlErr.Err != nil {
			// Recursively check the underlying error
			errType, _ := categorizeError(urlErr.Err)
			if errType != "" {
				return errType, ""
			}
		}
	}

	// Default to connection_error for unknown network errors
	if _, ok := err.(net.Error); ok {
		return "connection_error", ""
	}

	// Unknown error type
	return "unknown_error", ""
}

func makeRequest(client *http.Client, targetURL, proxyName, proxyType string) {
	start := time.Now()

	resp, err := client.Get(targetURL)
	duration := time.Since(start).Seconds()

	if err != nil {
		// Categorize error
		errorType, _ := categorizeError(err)
		requestsTotal.WithLabelValues(proxyName, proxyType, "error", errorType).Inc()
		requestDuration.WithLabelValues(proxyName, proxyType).Observe(duration)
		log.Printf("[%s] Error making request to %s: %v", proxyName, targetURL, err)
		return
	}
	defer resp.Body.Close()

	// Read and discard response body to free up connection
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		// Error reading response body
		requestsTotal.WithLabelValues(proxyName, proxyType, "error", "read_error").Inc()
		requestDuration.WithLabelValues(proxyName, proxyType).Observe(duration)
		log.Printf("[%s] Error reading response: %v", proxyName, err)
		return
	}

	// Check HTTP status code
	if resp.StatusCode >= 400 {
		errorType := "http_" + strconv.Itoa(resp.StatusCode)
		requestsTotal.WithLabelValues(proxyName, proxyType, "error", errorType).Inc()
		requestDuration.WithLabelValues(proxyName, proxyType).Observe(duration)
		log.Printf("[%s] HTTP error %d for request to %s", proxyName, resp.StatusCode, targetURL)
		return
	}

	// Success
	requestsTotal.WithLabelValues(proxyName, proxyType, "success", "").Inc()
	requestDuration.WithLabelValues(proxyName, proxyType).Observe(duration)
}

// runProxy starts a proxy runner that sends requests at specified interval
func runProxy(proxyConfig Proxy, targetURL string, requestInterval, requestTimeout time.Duration) {
	// Create transport for this proxy
	transport, err := createProxyTransport(proxyConfig.Type, proxyConfig.URL)
	if err != nil {
		log.Fatalf("[%s] Error creating proxy transport: %v", proxyConfig.Name, err)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
	}

	log.Printf("[%s] Starting proxy runner (type: %s, url: %s)", proxyConfig.Name, proxyConfig.Type, maskAuth(proxyConfig.URL))

	// Create ticker for this proxy
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()

	// Send initial request immediately
	go makeRequest(client, targetURL, proxyConfig.Name, proxyConfig.Type)

	// Send requests at intervals
	for range ticker.C {
		go makeRequest(client, targetURL, proxyConfig.Name, proxyConfig.Type)
	}
}

func main() {
	// Load YAML config
	config, err := loadProxyConfig()
	if err != nil {
		log.Fatalf("Error loading proxy configuration: %v", err)
	}

	// Initialize requestDuration metric with configured buckets
	buckets := getLatencyBuckets(config)
	initRequestDurationMetric(buckets)
	log.Printf("Using latency buckets: %v", buckets)

	targetURL := config.TargetURL
	requestInterval := time.Duration(config.RequestInterval) * time.Millisecond
	requestTimeout := time.Duration(config.RequestTimeout) * time.Second

	// Default metrics port to 8080 if not specified
	metricsPort := config.MetricsPort
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
	log.Printf("  Number of proxies: %d", len(config.Proxies))

	// Start each proxy in a separate goroutine
	for _, proxyConfig := range config.Proxies {
		go runProxy(proxyConfig, targetURL, requestInterval, requestTimeout)
	}

	// Keep main goroutine alive
	select {}
}
