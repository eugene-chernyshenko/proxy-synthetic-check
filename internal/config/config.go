package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

// ProxyConfig represents the YAML configuration file structure
type ProxyConfig struct {
	DefaultTargetURL string    `yaml:"default_target_url"`
	RequestInterval  int       `yaml:"request_interval_ms"`
	RequestTimeout   int       `yaml:"request_timeout"`
	MetricsPort      int       `yaml:"metrics_port"`
	LatencyBuckets   []float64 `yaml:"latency_buckets,omitempty"` // Optional custom buckets
	Proxies          []Proxy   `yaml:"proxies"`
}

// Proxy represents a single proxy configuration
type Proxy struct {
	Protocol  string            `yaml:"protocol"`             // socks5, http
	Proxy     string            `yaml:"proxy"`                // username:password@host:port or host:port (no scheme)
	TargetURL string            `yaml:"target_url,omitempty"` // Optional target URL (overrides default)
	Labels    map[string]string `yaml:"labels"`               // Custom labels for metrics
}

// GetTargetURL returns the target URL for this proxy, using proxy-specific URL if set,
// otherwise falling back to the default from config
func (p *Proxy) GetTargetURL(defaultURL string) string {
	if p.TargetURL != "" {
		return p.TargetURL
	}
	return defaultURL
}

// Load reads and parses the configuration from proxies.yaml file
func Load() (*ProxyConfig, error) {
	data, err := os.ReadFile("proxies.yaml")
	if err != nil {
		return nil, err
	}

	var cfg ProxyConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if len(cfg.Proxies) == 0 {
		return nil, errors.New("no proxies configured in config file")
	}

	return &cfg, nil
}

// GetLatencyBuckets returns latency buckets, using config if provided, otherwise defaults
func (c *ProxyConfig) GetLatencyBuckets() []float64 {
	if len(c.LatencyBuckets) > 0 {
		return c.LatencyBuckets
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
		10.0, // 10s - timeout
	}
}
