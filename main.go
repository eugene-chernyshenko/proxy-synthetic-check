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

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/proxy"
)

var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "requests_total",
			Help: "Total number of requests",
		},
		[]string{"status", "error_type", "http_status_code"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "Request latency distribution",
			Buckets: prometheus.DefBuckets,
		},
		[]string{},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(requestDuration)
}

// hasScheme checks if a string contains a URL scheme using url.Parse
func hasScheme(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme != ""
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

func loadEnv() {
	// Load .env file if it exists (for dev environment)
	// Ignore error if file is not found
	_ = godotenv.Load()
}

func getProxyURL() (string, error) {
	// Try to get from environment variable first
	proxyURL := os.Getenv("SOCKS5_PROXY")
	if proxyURL == "" {
		return "", errors.New("SOCKS5_PROXY is not set. Set SOCKS5_PROXY environment variable or create .env file")
	}
	return proxyURL, nil
}

func getTargetURL() (string, error) {
	targetURL := os.Getenv("TARGET_URL")
	if targetURL == "" {
		return "", errors.New("TARGET_URL is not set. Set TARGET_URL environment variable or create .env file")
	}
	return targetURL, nil
}

func getRequestInterval() (time.Duration, error) {
	intervalStr := os.Getenv("REQUEST_INTERVAL_MS")
	if intervalStr == "" {
		return 0, errors.New("REQUEST_INTERVAL_MS is not set. Set REQUEST_INTERVAL_MS environment variable or create .env file")
	}
	intervalMs, err := strconv.Atoi(intervalStr)
	if err != nil {
		return 0, errors.New("REQUEST_INTERVAL_MS must be a valid integer (milliseconds)")
	}
	if intervalMs <= 0 {
		return 0, errors.New("REQUEST_INTERVAL_MS must be greater than 0")
	}
	return time.Duration(intervalMs) * time.Millisecond, nil
}

func getRequestTimeout() (time.Duration, error) {
	timeoutStr := os.Getenv("REQUEST_TIMEOUT")
	if timeoutStr == "" {
		// Default timeout 30 seconds
		return 30 * time.Second, nil
	}
	timeout, err := strconv.Atoi(timeoutStr)
	if err != nil {
		return 0, errors.New("REQUEST_TIMEOUT must be a valid integer (seconds)")
	}
	if timeout <= 0 {
		return 0, errors.New("REQUEST_TIMEOUT must be greater than 0")
	}
	return time.Duration(timeout) * time.Second, nil
}

func getMetricsPort() int {
	portStr := os.Getenv("METRICS_PORT")
	if portStr == "" {
		return 8080
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		log.Printf("Invalid METRICS_PORT, using default 8080")
		return 8080
	}
	return port
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

func makeRequest(client *http.Client, targetURL string) {
	start := time.Now()

	resp, err := client.Get(targetURL)
	duration := time.Since(start).Seconds()

	if err != nil {
		// Categorize error
		errorType, _ := categorizeError(err)
		requestsTotal.WithLabelValues("error", errorType, "").Inc()
		requestDuration.WithLabelValues().Observe(duration)
		log.Printf("Error making request to %s: %v", targetURL, err)
		return
	}
	defer resp.Body.Close()

	// Read and discard response body to free up connection
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		// Error reading response body
		requestsTotal.WithLabelValues("error", "read_error", "").Inc()
		requestDuration.WithLabelValues().Observe(duration)
		log.Printf("Error reading response: %v", err)
		return
	}

	// Check HTTP status code
	if resp.StatusCode >= 400 {
		statusCode := strconv.Itoa(resp.StatusCode)
		requestsTotal.WithLabelValues("error", "http_error", statusCode).Inc()
		requestDuration.WithLabelValues().Observe(duration)
		log.Printf("HTTP error %d for request to %s", resp.StatusCode, targetURL)
		return
	}

	// Success
	requestsTotal.WithLabelValues("success", "", "").Inc()
	requestDuration.WithLabelValues().Observe(duration)
}

func main() {
	// Load environment variables from .env file (if exists)
	loadEnv()

	// Get configuration from environment variables
	targetURL, err := getTargetURL()
	if err != nil {
		log.Fatalf("Error getting target URL: %v", err)
	}

	proxyURL, err := getProxyURL()
	if err != nil {
		log.Fatalf("Error getting proxy: %v", err)
	}

	requestInterval, err := getRequestInterval()
	if err != nil {
		log.Fatalf("Error getting request interval: %v", err)
	}

	requestTimeout, err := getRequestTimeout()
	if err != nil {
		log.Fatalf("Error getting request timeout: %v", err)
	}

	// Add socks5:// scheme if not specified
	if !hasScheme(proxyURL) {
		proxyURL = "socks5://" + proxyURL
	}

	metricsPort := getMetricsPort()

	log.Printf("Configuration:")
	log.Printf("  Target URL: %s", targetURL)
	log.Printf("  SOCKS5 proxy: %s", maskAuth(proxyURL))
	log.Printf("  Request interval: %v", requestInterval)
	log.Printf("  Request timeout: %v", requestTimeout)
	log.Printf("  Metrics port: %d", metricsPort)

	// Parse proxy URL
	proxyURI, err := url.Parse(proxyURL)
	if err != nil {
		log.Fatalf("Error parsing proxy URL: %v", err)
	}

	// Extract proxy address (host:port)
	proxyAddr := proxyURI.Host
	if proxyAddr == "" {
		log.Fatal("Error: proxy address (host:port) is not specified")
	}

	// Extract authentication credentials
	var auth *proxy.Auth
	if proxyURI.User != nil {
		password, _ := proxyURI.User.Password()
		auth = &proxy.Auth{
			User:     proxyURI.User.Username(),
			Password: password,
		}
	}

	// Create SOCKS5 dialer
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
	if err != nil {
		log.Fatalf("Error creating SOCKS5 dialer: %v", err)
	}

	// Create HTTP transport with SOCKS5 dialer
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
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

	log.Printf("Starting to send requests every %v...", requestInterval)

	// Create ticker for sending requests at specified interval
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()

	// Send initial request immediately
	go makeRequest(client, targetURL)

	// Send requests at intervals in separate goroutines
	for range ticker.C {
		go makeRequest(client, targetURL)
	}
}
