package request

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"eugene-chernyshenko/proxy-synthetic-check/internal/metrics"
)

// Make performs HTTP request and records metrics
func Make(m *metrics.Metrics, client *http.Client, targetURL, proxyID, proxyProtocol string, labels map[string]string) {
	start := time.Now()

	resp, err := client.Get(targetURL)
	duration := time.Since(start).Seconds()

	// Build label values: proxy_id, proxy_protocol, ...labelKeys..., status, error
	buildLabelValues := func(status, errorValue string) []string {
		values := []string{proxyID, proxyProtocol}
		for _, key := range m.LabelKeys {
			if val, ok := labels[key]; ok {
				values = append(values, val)
			} else {
				values = append(values, "")
			}
		}
		values = append(values, status, errorValue)
		return values
	}

	// Build label values for duration: proxy_id, proxy_protocol, ...labelKeys...
	buildDurationLabelValues := func() []string {
		values := []string{proxyID, proxyProtocol}
		for _, key := range m.LabelKeys {
			if val, ok := labels[key]; ok {
				values = append(values, val)
			} else {
				values = append(values, "")
			}
		}
		return values
	}

	if err != nil {
		// Categorize error
		errorType, _ := CategorizeError(err)
		m.RequestsTotal.WithLabelValues(buildLabelValues("error", errorType)...).Inc()
		m.RequestDuration.WithLabelValues(buildDurationLabelValues()...).Observe(duration)
		log.Printf("[%s] Error making request to %s: %v", proxyID, targetURL, err)
		return
	}
	defer resp.Body.Close()

	// Read and discard response body to free up connection
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		// Error reading response body
		m.RequestsTotal.WithLabelValues(buildLabelValues("error", "read_error")...).Inc()
		m.RequestDuration.WithLabelValues(buildDurationLabelValues()...).Observe(duration)
		log.Printf("[%s] Error reading response: %v", proxyID, err)
		return
	}

	// Check HTTP status code
	if resp.StatusCode >= 400 {
		errorType := "http_" + strconv.Itoa(resp.StatusCode)
		m.RequestsTotal.WithLabelValues(buildLabelValues("error", errorType)...).Inc()
		m.RequestDuration.WithLabelValues(buildDurationLabelValues()...).Observe(duration)
		log.Printf("[%s] HTTP error %d for request to %s", proxyID, resp.StatusCode, targetURL)
		return
	}

	// Success
	m.RequestsTotal.WithLabelValues(buildLabelValues("success", "")...).Inc()
	m.RequestDuration.WithLabelValues(buildDurationLabelValues()...).Observe(duration)
}

// CategorizeError categorizes errors into types for metrics
func CategorizeError(err error) (errorType, httpStatusCode string) {
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
			errType, _ := CategorizeError(urlErr.Err)
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

