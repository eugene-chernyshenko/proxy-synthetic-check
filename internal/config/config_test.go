package config

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGetLatencyBuckets_WithCustomBuckets(t *testing.T) {
	cfg := &ProxyConfig{
		LatencyBuckets: []float64{0.1, 0.5, 1.0},
	}

	buckets := cfg.GetLatencyBuckets()

	expected := []float64{0.1, 0.5, 1.0}
	if !reflect.DeepEqual(buckets, expected) {
		t.Errorf("GetLatencyBuckets() = %v, want %v", buckets, expected)
	}
}

func TestGetLatencyBuckets_WithDefaultBuckets(t *testing.T) {
	cfg := &ProxyConfig{}

	buckets := cfg.GetLatencyBuckets()

	expected := []float64{0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.75, 1.0, 1.5, 2.0, 5.0, 10.0}
	if !reflect.DeepEqual(buckets, expected) {
		t.Errorf("GetLatencyBuckets() = %v, want %v", buckets, expected)
	}
}

func TestParseYAML_Success(t *testing.T) {
	configContent := `
target_url: https://example.com
request_interval_ms: 1000
request_timeout: 30
metrics_port: 8080
proxies:
  - protocol: socks5
    proxy: user:pass@proxy.example.com:1080
    labels:
      name: test
      region: us
  - protocol: http
    proxy: proxy2.example.com:8080
    labels:
      name: test2
`

	var cfg ProxyConfig
	err := yaml.Unmarshal([]byte(configContent), &cfg)
	if err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if cfg.TargetURL != "https://example.com" {
		t.Errorf("TargetURL = %v, want https://example.com", cfg.TargetURL)
	}
	if cfg.RequestInterval != 1000 {
		t.Errorf("RequestInterval = %v, want 1000", cfg.RequestInterval)
	}
	if cfg.RequestTimeout != 30 {
		t.Errorf("RequestTimeout = %v, want 30", cfg.RequestTimeout)
	}
	if cfg.MetricsPort != 8080 {
		t.Errorf("MetricsPort = %v, want 8080", cfg.MetricsPort)
	}
	if len(cfg.Proxies) != 2 {
		t.Errorf("Proxies length = %v, want 2", len(cfg.Proxies))
	}

	// Test first proxy
	proxy1 := cfg.Proxies[0]
	if proxy1.Protocol != "socks5" {
		t.Errorf("Proxy1 Protocol = %v, want socks5", proxy1.Protocol)
	}
	if proxy1.Proxy != "user:pass@proxy.example.com:1080" {
		t.Errorf("Proxy1 Proxy = %v, want user:pass@proxy.example.com:1080", proxy1.Proxy)
	}
	if proxy1.Labels["name"] != "test" {
		t.Errorf("Proxy1 Labels[name] = %v, want test", proxy1.Labels["name"])
	}
	if proxy1.Labels["region"] != "us" {
		t.Errorf("Proxy1 Labels[region] = %v, want us", proxy1.Labels["region"])
	}

	// Test second proxy
	proxy2 := cfg.Proxies[1]
	if proxy2.Protocol != "http" {
		t.Errorf("Proxy2 Protocol = %v, want http", proxy2.Protocol)
	}
	if proxy2.Proxy != "proxy2.example.com:8080" {
		t.Errorf("Proxy2 Proxy = %v, want proxy2.example.com:8080", proxy2.Proxy)
	}
	if proxy2.Labels["name"] != "test2" {
		t.Errorf("Proxy2 Labels[name] = %v, want test2", proxy2.Labels["name"])
	}
}

func TestParseYAML_EmptyProxies(t *testing.T) {
	configContent := `
target_url: https://example.com
request_interval_ms: 1000
request_timeout: 30
proxies: []
`

	var cfg ProxyConfig
	err := yaml.Unmarshal([]byte(configContent), &cfg)
	if err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if len(cfg.Proxies) != 0 {
		t.Errorf("Proxies length = %v, want 0", len(cfg.Proxies))
	}
}

func TestParseYAML_WithLatencyBuckets(t *testing.T) {
	configContent := `
target_url: https://example.com
request_interval_ms: 1000
request_timeout: 30
latency_buckets: [0.1, 0.5, 1.0, 2.0]
proxies:
  - protocol: socks5
    proxy: proxy.example.com:1080
`

	var cfg ProxyConfig
	err := yaml.Unmarshal([]byte(configContent), &cfg)
	if err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	expected := []float64{0.1, 0.5, 1.0, 2.0}
	if !reflect.DeepEqual(cfg.LatencyBuckets, expected) {
		t.Errorf("LatencyBuckets = %v, want %v", cfg.LatencyBuckets, expected)
	}
}
