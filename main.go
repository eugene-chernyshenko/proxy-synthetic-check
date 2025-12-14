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

func main() {
	if len(os.Args) < 2 {
		log.Printf("Usage: go run main.go <target_url>")
		log.Printf("Example: go run main.go https://httpbin.org/ip")
		log.Printf("Proxy is taken from SOCKS5_PROXY environment variable or .env file")
		os.Exit(1)
	}

	// Load environment variables from .env file (if exists)
	loadEnv()

	targetURL := os.Args[1]

	// Get proxy URL from environment variables
	proxyURL, err := getProxyURL()
	if err != nil {
		log.Fatalf("Error getting proxy: %v", err)
	}

	// Add socks5:// scheme if not specified
	if !hasScheme(proxyURL) {
		proxyURL = "socks5://" + proxyURL
	}

	log.Printf("Using SOCKS5 proxy: %s", maskAuth(proxyURL))

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
		Timeout:   30 * time.Second,
	}

	// Execute request
	log.Printf("Making request to: %s", targetURL)
	resp, err := client.Get(targetURL)
	if err != nil {
		log.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	// Read and discard response body to free up connection
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}
}
