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
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/net/proxy"
)

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

func makeRequest(client *http.Client, targetURL string) {
	resp, err := client.Get(targetURL)
	if err != nil {
		log.Printf("Error making request to %s: %v", targetURL, err)
		return
	}
	defer resp.Body.Close()

	// Read and discard response body to free up connection
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		log.Printf("Error reading response: %v", err)
		return
	}
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

	log.Printf("Configuration:")
	log.Printf("  Target URL: %s", targetURL)
	log.Printf("  SOCKS5 proxy: %s", maskAuth(proxyURL))
	log.Printf("  Request interval: %v", requestInterval)
	log.Printf("  Request timeout: %v", requestTimeout)

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
