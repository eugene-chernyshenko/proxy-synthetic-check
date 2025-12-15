package metrics

import (
	"reflect"
	"testing"

	"eugene-chernyshenko/proxy-synthetic-check/internal/config"
)

func TestCollectLabelKeys_Logic(t *testing.T) {
	// Test the label collection logic by creating a single metrics instance
	// Note: We can't test New() multiple times due to Prometheus global registry,
	// so we test the label collection through one comprehensive test

	proxies := []config.Proxy{
		{
			Protocol: "socks5",
			Proxy:    "proxy1.example.com:1080",
			Labels: map[string]string{
				"name":   "wifi",
				"region": "us",
			},
		},
		{
			Protocol: "http",
			Proxy:    "proxy2.example.com:8080",
			Labels: map[string]string{
				"name":     "mobile",
				"region":   "eu",
				"provider": "provider-a",
			},
		},
		{
			Protocol: "socks5",
			Proxy:    "proxy3.example.com:1080",
			Labels: map[string]string{
				"name": "another",
			},
		},
		{
			Protocol: "http",
			Proxy:    "proxy4.example.com:8080",
			Labels:   nil, // Test nil labels
		},
		{
			Protocol: "socks5",
			Proxy:    "proxy5.example.com:1080",
			Labels:   map[string]string{}, // Test empty labels
		},
	}

	buckets := []float64{0.1, 0.5, 1.0}
	m := New(proxies, buckets)

	// Check that label keys are collected, deduplicated, and sorted alphabetically
	// Empty/nil labels should not affect the result
	expectedKeys := []string{"name", "provider", "region"}
	if !reflect.DeepEqual(m.LabelKeys, expectedKeys) {
		t.Errorf("LabelKeys = %v, want %v", m.LabelKeys, expectedKeys)
	}

	// Check that metrics are initialized
	if m.RequestsTotal == nil {
		t.Error("RequestsTotal is nil")
	}
	if m.RequestDuration == nil {
		t.Error("RequestDuration is nil")
	}
}

// Note: We can't test New() multiple times in the same test run due to Prometheus
// global registry. The empty labels case is tested indirectly in TestCollectLabelKeys_Logic
// by ensuring that nil/empty labels don't cause issues when mixed with non-empty labels.
